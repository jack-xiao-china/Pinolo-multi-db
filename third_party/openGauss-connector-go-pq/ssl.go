package pq

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"io/ioutil"
	"math"
	"os"
	"strings"
	"time"
)

// Number of days before certificate expiration alarm notification
const certWarningDays int = 7
var expectTLSFileModes = []os.FileMode{ 0400, 0500, 0600 }

func checkTLSFileMode(config *Config, names ...string)error{
	if !config.shouldLog(LogLevelWarn) {
		return nil
	}

	for _,name := range names {
		f, err := os.Open(name)
		if err != nil {
			return err
		}

		if info, err := f.Stat(); err == nil {
			mode := info.Mode()
			matchFlag := false
			for _, m := range expectTLSFileModes {
				if m == mode {
					matchFlag = true
					break
				}
			}
			if ! matchFlag {
				config.Logger.Log(context.Background(), LogLevelWarn, "ssl file permission should be rw(600) or less,",
					map[string]interface{}{name+" current: ":mode})
			}
		}
		f.Close()
	}
	return nil
}

func checkCertByteExpire(config *Config, cert []byte, certName string) error {
	x509Cert, err := x509.ParseCertificate(cert)
	if err != nil {
		return err
	}
	if !x509Cert.NotAfter.After(time.Now()) {
		return errors.New("the certificate has expired")
	}
	d := x509Cert.NotAfter.Sub(time.Now())
	if d <= time.Hour * time.Duration(certWarningDays * 24) {
		config.Logger.Log(context.Background(), LogLevelWarn, "The certificate is about to expire,",
			map[string]interface{}{"left days:": math.Ceil(float64(d / time.Hour / 24)), "file name:":certName})
	}
	return nil
}

func checkCertExpire(config *Config, filePath string) error {
	if !config.shouldLog(LogLevelWarn) {
		return nil
	}
	certByte, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil
	}
	for len(certByte) > 0 {
		var block *pem.Block
		block, certByte = pem.Decode(certByte)
		if block == nil {
			break
		}
		if block.Type != "CERTIFICATE" || len(block.Headers) != 0 {
			continue
		}
		if err := checkCertByteExpire(config, block.Bytes, filePath); err != nil {
			return err
		}
	}
	return nil
}

// configTLS uses libpq's TLS parameters to construct  []*tls.Config. It is
// necessary to allow returning multiple TLS configs as sslmode "allow" and
// "prefer" allow fallback.
func configTLS(settings map[string]string, config *Config) ([]*tls.Config, error) {
	host := settings["host"]
	sslmode := settings["sslmode"]
	sslrootcert := settings["sslrootcert"]
	sslcert := settings["sslcert"]
	sslkey := settings["sslkey"]
	sslPassword := settings["sslpassword"]

	// Match libpq default behavior
	if sslmode == "" {
		sslmode = "prefer"
	}

	tlsConfig := &tls.Config{}
	switch sslmode {
	case "disable":
		return []*tls.Config{nil}, nil
	case "allow", "prefer":
		tlsConfig.InsecureSkipVerify = true
	case "require":
		// According to PostgreSQL documentation, if a root CA file exists,
		// the behavior of sslmode=require should be the same as that of verify-ca
		if sslrootcert != "" {
			goto nextCase
		}
		tlsConfig.InsecureSkipVerify = true
		break
	nextCase:
		fallthrough
	case "verify-ca":
		// Don't perform the default certificate verification because it
		// will verify the hostname. Instead, verify the server's
		// certificate chain ourselves in VerifyPeerCertificate and
		// ignore the server name. This emulates libpq's verify-ca
		// behavior.
		tlsConfig.InsecureSkipVerify = true
		tlsConfig.VerifyPeerCertificate = func(certificates [][]byte, _ [][]*x509.Certificate) error {
			certs := make([]*x509.Certificate, len(certificates))
			for i, asn1Data := range certificates {
				cert, err := x509.ParseCertificate(asn1Data)
				if err != nil {
					return errors.New("failed to parse certificate from server: " + err.Error())
				}
				certs[i] = cert
			}

			// Leave DNSName empty to skip hostname verification.
			opts := x509.VerifyOptions{
				Roots:         tlsConfig.RootCAs,
				Intermediates: x509.NewCertPool(),
			}
			// Skip the first cert because it's the leaf. All others
			// are intermediates.
			for _, cert := range certs[1:] {
				opts.Intermediates.AddCert(cert)
			}
			_, err := certs[0].Verify(opts)
			return err
		}
	case "verify-full":
		tlsConfig.ServerName = host
	default:
		return nil, errors.New("sslmode is invalid")
	}

	if sslrootcert != "" {
		caCertPool := x509.NewCertPool()

		caPath := sslrootcert
		if err := checkTLSFileMode(config, caPath); err !=nil{
			return nil, fmt.Errorf("unable to read CA file: %w", err)
		}
		caCert, err := ioutil.ReadFile(caPath)
		if err != nil {
			return nil, fmt.Errorf("unable to read CA file: %w", err)
		}
		if err := checkCertExpire(config, caPath); err != nil {
			return nil, fmt.Errorf("unable to check CA file: %w", err)
		}
		if !caCertPool.AppendCertsFromPEM(caCert) {
			return nil, errors.New("unable to add CA to cert pool")
		}

		tlsConfig.RootCAs = caCertPool
		tlsConfig.ClientCAs = caCertPool
	}

	if (sslcert != "" && sslkey == "") || (sslcert == "" && sslkey != "") {
		return nil, errors.New(`both "sslcert" and "sslkey" are required`)
	}

	var cert tls.Certificate
	if sslcert != "" && sslkey != "" {
		var err error
		var decodeSslPwd []byte
		if err = checkTLSFileMode(config, sslcert, sslkey); err !=nil{
			return nil, fmt.Errorf("unable to read cert file or sslkey file: %w", err)
		}
		if sslPassword == "" {
			cert, err = tls.LoadX509KeyPair(sslcert, sslkey)
		} else {
			decodeSslPwd,err = base64.StdEncoding.DecodeString(sslPassword)
			if err != nil {
				return nil, err
			}
			cert, err = loadX509KeyPairWithPassphrase(sslcert, sslkey, string(decodeSslPwd))
			clearBytes(decodeSslPwd)
		}
		if err != nil {
			return nil, fmt.Errorf("unable to read cert or key: %w", err)
		}
		if err = checkCertExpire(config, sslcert); err != nil {
			return nil, fmt.Errorf("unable to check certificate file: %w", err)
		}
	}

	tlsConfig.Certificates = []tls.Certificate{cert}
	tlsConfig.CipherSuites = []uint16{
		tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
		tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
		tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
		tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
	}
	tlsConfig.MinVersion = tls.VersionTLS12
	switch sslmode {
	case "allow":
		return []*tls.Config{nil, tlsConfig}, nil
	case "prefer":
		return []*tls.Config{tlsConfig, nil}, nil
	case "require", "verify-ca", "verify-full":
		return []*tls.Config{tlsConfig}, nil
	default:
		panic("BUG: bad sslmode should already have been caught")
	}
}

func loadX509KeyPairWithPassphrase(certFile, keyFile, passPhase string) (tls.Certificate, error) {
	certPEMBlock, err := ioutil.ReadFile(certFile)
	if err != nil {
		return tls.Certificate{}, err
	}
	keyPEMBlock, err := ioutil.ReadFile(keyFile)
	if err != nil {
		return tls.Certificate{}, err
	}
	return x509KeyPairWithPassphrase(certPEMBlock, keyPEMBlock, passPhase)
}

func x509KeyPairWithPassphrase(certPEMBlock, keyPEMBlock []byte, passPhrase string) (tls.Certificate, error) {
	fail := func(err error) (tls.Certificate, error) { return tls.Certificate{}, err }

	var cert tls.Certificate
	var skippedBlockTypes []string
	for {
		var certDERBlock *pem.Block = nil
		certDERBlock, certPEMBlock = pem.Decode(certPEMBlock)
		if certDERBlock == nil {
			break
		}
		if certDERBlock.Type == "CERTIFICATE" {
			cert.Certificate = append(cert.Certificate, certDERBlock.Bytes)
		} else {
			skippedBlockTypes = append(skippedBlockTypes, certDERBlock.Type)
		}
	}

	if len(cert.Certificate) == 0 {
		if len(skippedBlockTypes) == 0 {
			return fail(errors.New("tls: failed to find any PEM data in certificate input"))
		}
		if len(skippedBlockTypes) == 1 && strings.HasSuffix(skippedBlockTypes[0], "PRIVATE KEY") {
			return fail(errors.New("tls: failed to find certificate PEM data in certificate input, but did find a private key; PEM inputs may have been switched"))
		}
		return fail(fmt.Errorf("tls: failed to find \"CERTIFICATE\" PEM block in certificate input after skipping PEM blocks of the following types: %v", skippedBlockTypes))
	}

	skippedBlockTypes = skippedBlockTypes[:0]
	var keyDERBlock *pem.Block
	for {
		keyDERBlock, keyPEMBlock = pem.Decode(keyPEMBlock)
		if keyDERBlock == nil {
			if len(skippedBlockTypes) == 0 {
				return fail(errors.New("tls: failed to find any PEM data in key input"))
			}
			if len(skippedBlockTypes) == 1 && skippedBlockTypes[0] == "CERTIFICATE" {
				return fail(errors.New("tls: found a certificate rather than a key in the PEM for the private key"))
			}
			return fail(fmt.Errorf("tls: failed to find PEM block with type ending in \"PRIVATE KEY\" in key input after skipping PEM blocks of the following types: %v", skippedBlockTypes))
		}
		if keyDERBlock.Type == "PRIVATE KEY" || strings.HasSuffix(keyDERBlock.Type, " PRIVATE KEY") {
			break
		}
		skippedBlockTypes = append(skippedBlockTypes, keyDERBlock.Type)
	}

	// We don't need to parse the public key for TLS, but we so do anyway
	// to check that it looks sane and matches the private key.
	x509Cert, err := x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
		return fail(err)
	}

	cert.PrivateKey, err = decryptPEM(keyDERBlock,[]byte(passPhrase))
	if err != nil {
		return fail(err)
	}

	switch pub := x509Cert.PublicKey.(type) {
	case *rsa.PublicKey:
		priv, ok := cert.PrivateKey.(*rsa.PrivateKey)
		if !ok {
			return fail(errors.New("tls: private key type does not match public key type"))
		}
		if pub.N.Cmp(priv.N) != 0 {
			return fail(errors.New("tls: private key does not match public key"))
		}
	case *ecdsa.PublicKey:
		priv, ok := cert.PrivateKey.(*ecdsa.PrivateKey)
		if !ok {
			return fail(errors.New("tls: private key type does not match public key type"))
		}
		if pub.X.Cmp(priv.X) != 0 || pub.Y.Cmp(priv.Y) != 0 {
			return fail(errors.New("tls: private key does not match public key"))
		}
	default:
		return fail(errors.New("tls: unknown public key algorithm"))
	}

	return cert, nil
}

func derToPrivateKey(der []byte) (key interface{}, err error) {
	if key, err = x509.ParsePKCS1PrivateKey(der); err == nil {
		return key, nil
	}

	if key, err = x509.ParsePKCS8PrivateKey(der); err == nil {
		switch key.(type) {
		case *rsa.PrivateKey, *ecdsa.PrivateKey:
			return
		default:
			return nil, errors.New("Found unknown private key type in PKCS#8 wrapping")
		}
	}

	if key, err = x509.ParseECPrivateKey(der); err == nil {
		return
	}

	return nil, errors.New("Invalid key type. The DER must contain an rsa.PrivateKey or ecdsa.PrivateKey")
}

func decryptPEM(block *pem.Block, passPhrase []byte) (crypto.PrivateKey, error) {
	der, err := x509.DecryptPEMBlock(block, passPhrase)
	if err != nil {
		return nil, fmt.Errorf("Failed PEM decryption [%s]", err)
	}

	privateKey, err := derToPrivateKey(der)
	if err != nil {
		return nil, err
	}

	var raw []byte
	switch k := privateKey.(type) {
	case *ecdsa.PrivateKey:
		raw, err = x509.MarshalECPrivateKey(k)
		if err != nil {
			return nil, err
		}
	case *rsa.PrivateKey:
		raw = x509.MarshalPKCS1PrivateKey(k)
	default:
		return nil, errors.New("Invalid key type. It must be *ecdsa.PrivateKey or *rsa.PrivateKey")
	}

	rawBase64 := base64.StdEncoding.EncodeToString(raw)
	derBase64 := base64.StdEncoding.EncodeToString(der)
	if rawBase64 != derBase64 {
		return nil, errors.New("Invalid decrypt PEM: raw does not match with der")
	}
	return privateKey, nil
}