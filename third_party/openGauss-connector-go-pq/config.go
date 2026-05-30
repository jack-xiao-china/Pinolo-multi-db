package pq

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"errors"
	"fmt"
	"io/ioutil"
	"math"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Config is the settings used to establish a connection to a PostgreSQL server. It must be created by ParseConfig. A
// manually initialized Config will cause ConnectConfig to panic.
type Config struct {
	Host           string // host (e.g. localhost) or absolute path to unix domain socket directory (e.g. /private/tmp)
	Port           uint16
	Database       string
	User           string
	Password       string
	TLSConfig      *tls.Config // nil disables TLS
	ConnectTimeout time.Duration
	DialFunc       DialFunc   // e.g. net.Dialer.DialContext
	LookupFunc     LookupFunc // e.g. net.Resolver.LookupHost
	// BuildFrontend  BuildFrontendFunc
	RuntimeParams map[string]string // Run-time parameters to set on connection as session default values (e.g. search_path or application_name)
	Fallbacks     []*FallbackConfig
	crlList *pkix.CertificateList

	targetSessionAttrs uint8
	// ValidateConnect is called during a connection attempt after a successful authentication with the PostgreSQL server.
	// It can be used to validate that the server is acceptable. If this returns an error the connection is closed and the next
	// fallback config is tried. This allows implementing high availability behavior such as libpq does with target_session_attrs.
	// ValidateConnect ValidateConnectFunc

	// AfterConnect is called after ValidateConnect. It can be used to set up the connection (e.g. Set session variables
	// or prepare statements). If this returns an error the connection attempt fails.
	// AfterConnect AfterConnectFunc

	// OnNotice is a callback function called when a notice response is received.
	// OnNotice NoticeHandler

	// OnNotification is a callback function called when a notification from the LISTEN/NOTIFY system is received.
	// OnNotification NotificationHandler

	createdByParseConfig bool // Used to enforce created by ParseConfig rule.

	// If set, this connection should never use the binary format when
	// receiving query results from prepared statements.  Only provided for
	// debugging.
	disablePreparedBinaryResult bool
	binaryParameters bool

	Logger   Logger
	LogLevel LogLevel
}

// Copy returns a deep copy of the config that is safe to use and modify.
// The only exception is the TLSConfig field:
// according to the tls.Config docs it must not be modified after creation.
func (c *Config) Copy() *Config {
	newConf := new(Config)
	*newConf = *c
	if newConf.TLSConfig != nil {
		newConf.TLSConfig = c.TLSConfig.Clone()
	}
	if newConf.RuntimeParams != nil {
		newConf.RuntimeParams = make(map[string]string, len(c.RuntimeParams))
		for k, v := range c.RuntimeParams {
			newConf.RuntimeParams[k] = v
		}
	}
	if newConf.Fallbacks != nil {
		newConf.Fallbacks = make([]*FallbackConfig, len(c.Fallbacks))
		for i, fallback := range c.Fallbacks {
			newFallback := new(FallbackConfig)
			*newFallback = *fallback
			if newFallback.TLSConfig != nil {
				newFallback.TLSConfig = fallback.TLSConfig.Clone()
			}
			newConf.Fallbacks[i] = newFallback
		}
	}
	return newConf
}

func (c *Config) shouldLog(lvl LogLevel) bool {
	return c.Logger != nil && c.LogLevel >= lvl
}

func (c *Config) Log(ctx context.Context, level LogLevel, msg string, data map[string]interface{}) {
	if c.shouldLog(level) {
		c.Logger.Log(ctx, level, msg, data)
	}
}

// FallbackConfig is additional settings to attempt a connection with when the primary Config fails to establish a
// network connection. It is used for TLS fallback such as sslmode=prefer and high availability (HA) connections.
type FallbackConfig struct {
	Host      string // host (e.g. localhost) or path to unix domain socket directory (e.g. /private/tmp)
	Port      uint16
	TLSConfig *tls.Config // nil disables TLS
}

// NetworkAddress converts a PostgreSQL host and port into network and address suitable for use with
// net.Dial.
func NetworkAddress(host string, port uint16) (network, address string) {
	if strings.HasPrefix(host, "/") {
		network = "unix"
		address = filepath.Join(host, ".s.PGSQL.") + strconv.FormatInt(int64(port), 10)
	} else {
		network = "tcp"
		address = net.JoinHostPort(host, strconv.Itoa(int(port)))
	}
	return network, address
}

// ParseConfig builds a *Config with similar behavior to the PostgreSQL standard C library libpq. It uses the same
// defaults as libpq (e.g. port=5432) and understands most PG* environment variables. ParseConfig closely matches
// the parsing behavior of libpq. connString may either be in URL format or keyword = value format (DSN style).
// connString also may be empty to only read from the environment.
// If a password is not supplied it will attempt to read the .pgpass file.
//
//   # Example DSN
//   user=jack password=secret host=pg.example.com port=5432 dbname=mydb sslmode=verify-ca
//
//   # Example URL
//   postgres://jack:secret@pg.example.com:5432/mydb?sslmode=verify-ca
//
// The returned *Config may be modified. However, it is strongly recommended that any configuration that can be done
// through the connection string be done there. In particular the fields Host, Port, TLSConfig, and Fallbacks can be
// interdependent (e.g. TLSConfig needs knowledge of the host to validate the server certificate). These fields should
// not be modified individually. They should all be modified or all left unchanged.
//
// ParseConfig supports specifying multiple hosts in similar manner to libpq. Host and port may include comma separated
// values that will be tried in order. This can be used as part of a high availability system.
//
//   # Example URL
//   postgres://jack:secret@foo.example.com:5432,bar.example.com:5432/mydb
//
// ParseConfig currently recognizes the following environment variable and their parameter key word equivalents passed
// via database URL or DSN:
//
// 	 PGHOST
// 	 PGPORT
// 	 PGDATABASE
// 	 PGUSER
// 	 PGPASSWORD
// 	 PGSSLMODE
// 	 PGSSLCERT
// 	 PGSSLKEY
// 	 PGSSLROOTCERT
// 	 PGAPPNAME
// 	 PGCONNECT_TIMEOUT
// 	 PGTARGETSESSIONATTRS
// They are usually but not always the environment variable name downcased and without the "PG" prefix.
//
// Important Security Notes:
//
// ParseConfig tries to match libpq behavior with regard to PGSSLMODE. This includes defaulting to "prefer" behavior if
// not set.
//
// The sslmode "prefer" (the default), sslmode "allow", and multiple hosts are implemented via the Fallbacks field of
// the Config struct. If TLSConfig is manually changed it will not affect the fallbacks. For example, in the case of
// sslmode "prefer" this means it will first try the main Config settings which use TLS, then it will try the fallback
// which does not use TLS. This can lead to an unexpected unencrypted connection if the main TLS config is manually
// changed later but the unencrypted fallback is present. Ensure there are no stale fallbacks when manually setting
// TLCConfig.
//
// Other known differences with libpq:
//
// If a host name resolves into multiple addresses, libpq will try all addresses. pgconn will only try the first.
//
// When multiple hosts are specified, libpq allows them to have different passwords set via the .pgpass file. pgconn
// does not.
//
// In addition, ParseConfig accepts the following options:
//
// 	min_read_buffer_size
// 		The minimum size of the internal read buffer. Default 8192.
func clearBytes(bs []byte) {
	for i := 0; i < len(bs); i++ {
		bs[i] = 0
	}
}

func encodePassword(settings *map[string]string){
	pwdNames := []string {
		"password",
		"sslpassword",
	}
	for _, name := range pwdNames {
		if val,ok :=(*settings)[name]; ok {
			if val != "" {
				(*settings)[name]=base64.StdEncoding.EncodeToString([]byte(val))
			}
		}
	}
}

func ParseConfig(connString string) (*Config, error) {
	defaultSettings := defaultSettings()
	envSettings := parseEnvSettings()

	connStringSettings := make(map[string]string)
	if connString != "" {
		var err error
		// connString may be a database URL or a DSN
		if strings.HasPrefix(connString, "postgres://") || strings.HasPrefix(connString, "postgresql://") ||
			strings.HasPrefix(connString, "opengauss://") || strings.HasPrefix(connString, "mogdb://") {
			connStringSettings, err = parseURLSettings(connString)
			if err != nil {
				return nil, &parseConfigError{connString: connString, msg: "failed to parse as URL", err: err}
			}
		} else {
			connStringSettings, err = parseDSNSettings(connString)
			if err != nil {
				return nil, &parseConfigError{connString: connString, msg: "failed to parse as DSN", err: err}
			}
		}
	}

	settings := mergeSettings(defaultSettings, envSettings, connStringSettings)
	encodePassword(&settings)
	config := &Config{
		createdByParseConfig: true,
		Database:             settings["database"],
		User:                 settings["user"],
		Password:             settings["password"],
		RuntimeParams:        make(map[string]string),
	}

	if connectTimeoutSetting, present := settings["connect_timeout"]; present {
		connectTimeout, err := parseConnectTimeoutSetting(connectTimeoutSetting)
		if err != nil {
			return nil, &parseConfigError{connString: connString, msg: "invalid connect_timeout", err: err}
		}
		config.ConnectTimeout = connectTimeout
		config.DialFunc = makeConnectTimeoutDialFunc(connectTimeout)
	} else {
		defaultDialer := makeDefaultDialer()
		config.DialFunc = defaultDialer.DialContext
	}

	config.LookupFunc = makeDefaultResolver().LookupHost

	notRuntimeParams := map[string]struct{}{
		"host":                           struct{}{},
		"port":                           struct{}{},
		"database":                       struct{}{},
		"user":                           struct{}{},
		"password":                       struct{}{},
		"connect_timeout":                struct{}{},
		"sslmode":                        struct{}{},
		"sslkey":                         struct{}{},
		"sslpassword":                    struct{}{},
		"sslcert":                        struct{}{},
		"sslrootcert":                    struct{}{},
		"sslcrl":                         struct{}{},
		"target_session_attrs":           struct{}{},
		"min_read_buffer_size":           struct{}{},
		"disable_prepared_binary_result": struct{}{},
		"binary_parameters":              struct{}{},
		"loggerLevel":                    struct{}{},
	}

	for k, v := range settings {
		if _, present := notRuntimeParams[k]; present {
			continue
		}
		config.RuntimeParams[k] = v
	}
	if loggerLevel, ok := settings["loggerLevel"]; ok {
		var err error
		config.LogLevel, err = LogLevelFromString(strings.ToLower(loggerLevel))
		if err != nil {
			return nil, &parseConfigError{connString: connString, msg: "invalid loggerLevel", err: err}
		}
		if config.Logger == nil {
			config.Logger = NewPrintfLogger(config.LogLevel)
		}
	}

	fallbacks := []*FallbackConfig{}

	hosts := strings.Split(settings["host"], ",")
	ports := strings.Split(settings["port"], ",")

	for i, host := range hosts {
		var portStr string
		if i < len(ports) {
			portStr = ports[i]
		} else {
			portStr = ports[0]
		}

		port, err := parsePort(portStr)
		if err != nil {
			return nil, &parseConfigError{connString: connString, msg: "invalid port", err: err}
		}

		var tlsConfigs []*tls.Config

		// Ignore TLS settings if Unix domain socket like libpq
		if network, _ := NetworkAddress(host, port); network == "unix" {
			tlsConfigs = append(tlsConfigs, nil)
		} else {
			var err error
			tlsConfigs, err = configTLS(settings, config)
			if err != nil {
				return nil, &parseConfigError{connString: connString, msg: "failed to configure TLS", err: err}
			}
		}

		for _, tlsConfig := range tlsConfigs {
			fallbacks = append(fallbacks, &FallbackConfig{
				Host:      host,
				Port:      port,
				TLSConfig: tlsConfig,
			})
		}
	}

	config.Host = fallbacks[0].Host
	config.Port = fallbacks[0].Port
	config.TLSConfig = fallbacks[0].TLSConfig
	config.Fallbacks = fallbacks[1:]

	tryParseSslCrl(settings, config)
	targetSessionAttrs, err := parseTargetSessionAttr(settings, connString)
	if err != nil {
		return nil, err  // no need to raise parseConfigError
	}

	config.targetSessionAttrs = targetSessionAttrs

	config.disablePreparedBinaryResult, err = parseBoolSettings("disable_prepared_binary_result",settings,false)
	if err != nil{
		return nil, &parseConfigError{connString: connString, msg: "invalid disable_prepared_binary_result", err: err}
	}
	config.binaryParameters, err = parseBoolSettings("binary_parameters", settings,false)
	if err != nil{
		return nil, &parseConfigError{connString: connString, msg: "invalid binary_parameters", err: err}
	}
	return config, nil
}

func tryParseSslCrl(settings map[string]string, config *Config) {
	sslCrl := settings["sslcrl"]
	var crlList *pkix.CertificateList
	if sslCrl != "" {
		if err := checkTLSFileMode(config, sslCrl); err !=nil{
			config.Log(context.Background(), LogLevelError,"unable to read crl file,",map[string]interface{}{"error info:":err.Error()})
			return
		}

		pemBytes, err := ioutil.ReadFile(sslCrl)
		if err != nil {
			config.Log(context.Background(), LogLevelError,"failed to read file,",map[string]interface{}{"error info:":err.Error()})
			return
		}
		crlList, err = x509.ParseCRL(pemBytes)
		if err != nil {
			config.Log(context.Background(), LogLevelError,"failed to parse crl,",map[string]interface{}{"error info:":err.Error()})
			return
		}

		if !crlList.TBSCertList.NextUpdate.After(time.Now()) {
			config.Log(context.Background(), LogLevelError,"the crl has expired,",map[string]interface{}{})
			return
		}

		d := crlList.TBSCertList.NextUpdate.Sub(time.Now())
		if d <= time.Hour * time.Duration(certWarningDays * 24) {
			config.Log(context.Background(), LogLevelWarn, "The crl is about to expire,",
				map[string]interface{}{"left days:": math.Ceil(float64(d / time.Hour / 24)), "file name:": sslCrl})
		}
		config.crlList = crlList
	}
	return
}

func parseBoolSettings(key string,settings map[string]string, defaultVal bool) (val bool, err error) {
	val = defaultVal
	if value, ok := settings[key]; ok {
		if value == "yes" {
			val = true
		} else if value == "no" {
			val = false
		} else if value != "" {
			return val, fmt.Errorf("unrecognized value %q for %s", value, key)
		}
	}
	return val, nil
}

func mergeSettings(settingSets ...map[string]string) map[string]string {
	settings := make(map[string]string)

	for _, s2 := range settingSets {
		for k, v := range s2 {
			settings[k] = v
		}
	}

	return settings
}

func parseTargetSessionAttr(settings map[string]string, connString string)(uint8, error){
	if _, ok := settings["target_session_attrs"]; !ok{
		return targetSessionAttrsAny, nil
	}
	switch settings["target_session_attrs"] {
	case "any","":
		return targetSessionAttrsAny, nil
	case "master":
		return targetSessionAttrsMaster, nil
	case "slave":
		return targetSessionAttrsSlave, nil
	case "preferSlave":
		return targetSessionAttrsPreferSlave, nil
	case "read-write":
		return targetSessionAttrsReadWrite, nil
	case "read-only":
		return targetSessionAttrsReadOnly, nil
	default:
		return 0, &parseConfigError{connString: connString, msg: fmt.Sprintf("unknown target_session_attrs value: %v", settings["target_session_attrs"])}
	}
}

func convertTargetSessionAttrToString(targetSessionAttr uint8)string{
	switch targetSessionAttr {
	case targetSessionAttrsAny:
		return "any"
	case targetSessionAttrsMaster:
		return "master"
	case targetSessionAttrsSlave:
		return "slave"
	case targetSessionAttrsPreferSlave:
		return "preferSlave"
	case targetSessionAttrsReadWrite:
		return "read-write"
	case targetSessionAttrsReadOnly:
		return "read-only"
	default:
		return ""
	}
}

func parseEnvSettings() map[string]string {
	settings := make(map[string]string)

	nameMap := map[string]string{
		"PGHOST":               "host",
		"PGPORT":               "port",
		"PGDATABASE":           "database",
		"PGUSER":               "user",
		"PGAPPNAME":            "application_name",
		"PGCONNECT_TIMEOUT":    "connect_timeout",
		"PGSSLMODE":            "sslmode",
		"PGSSLKEY":             "sslkey",
		"PGSSLCERT":            "sslcert",
		"PGSSLROOTCERT":        "sslrootcert",
		"PGSSLCRL":             "sslcrl",
		"PGTARGETSESSIONATTRS": "target_session_attrs",
		"PGLOGGERLEVEL":        "loggerLevel",
	}

	for envname, realname := range nameMap {
		value := os.Getenv(envname)
		if value != "" {
			settings[realname] = value
		}
	}

	return settings
}

func parseURLSettings(connString string) (map[string]string, error) {
	settings := make(map[string]string)

	url, err := url.Parse(connString)
	if err != nil {
		return nil, err
	}

	if url.User != nil {
		settings["user"] = url.User.Username()
		if password, present := url.User.Password(); present {
			settings["password"] = password
		}
	}

	// Handle multiple host:port's in url.Host by splitting them into host,host,host and port,port,port.
	var hosts []string
	var ports []string
	for _, host := range strings.Split(url.Host, ",") {
		if host == "" {
			continue
		}
		if isIPOnly(host) {
			hosts = append(hosts, strings.Trim(host, "[]"))
			continue
		}
		h, p, err := net.SplitHostPort(host)
		if err != nil {
			return nil, fmt.Errorf("failed to split host:port in '%s', err: %w", host, err)
		}
		hosts = append(hosts, h)
		ports = append(ports, p)
	}
	if len(hosts) > 0 {
		settings["host"] = strings.Join(hosts, ",")
	}
	if len(ports) > 0 {
		settings["port"] = strings.Join(ports, ",")
	}

	database := strings.TrimLeft(url.Path, "/")
	if database != "" {
		settings["database"] = database
	}

	for k, v := range url.Query() {
		settings[k] = v[0]
	}

	return settings, nil
}

func isIPOnly(host string) bool {
	return net.ParseIP(strings.Trim(host, "[]")) != nil || !strings.Contains(host, ":")
}

var asciiSpace = [256]uint8{'\t': 1, '\n': 1, '\v': 1, '\f': 1, '\r': 1, ' ': 1}

func parseDSNSettings(s string) (map[string]string, error) {
	settings := make(map[string]string)

	nameMap := map[string]string{
		"dbname": "database",
	}

	for len(s) > 0 {
		var key, val string
		eqIdx := strings.IndexRune(s, '=')
		if eqIdx < 0 {
			return nil, errors.New("invalid dsn")
		}

		key = strings.Trim(s[:eqIdx], " \t\n\r\v\f")
		s = strings.TrimLeft(s[eqIdx+1:], " \t\n\r\v\f")
		if len(s) == 0 {
		} else if s[0] != '\'' {
			end := 0
			for ; end < len(s); end++ {
				if asciiSpace[s[end]] == 1 {
					break
				}
				if s[end] == '\\' {
					end++
					if end == len(s) {
						return nil, errors.New("invalid backslash")
					}
				}
			}
			val = strings.Replace(strings.Replace(s[:end], "\\\\", "\\", -1), "\\'", "'", -1)
			if end == len(s) {
				s = ""
			} else {
				s = s[end+1:]
			}
		} else { // quoted string
			s = s[1:]
			end := 0
			for ; end < len(s); end++ {
				if s[end] == '\'' {
					break
				}
				if s[end] == '\\' {
					end++
				}
			}
			if end == len(s) {
				return nil, errors.New("unterminated quoted string in connection info string")
			}
			val = strings.Replace(strings.Replace(s[:end], "\\\\", "\\", -1), "\\'", "'", -1)
			if end == len(s) {
				s = ""
			} else {
				s = s[end+1:]
			}
		}

		if k, ok := nameMap[key]; ok {
			key = k
		}

		if key == "" {
			return nil, errors.New("invalid dsn")
		}

		settings[key] = val
	}

	return settings, nil
}

func parsePort(s string) (uint16, error) {
	port, err := strconv.ParseUint(s, 10, 16)
	if err != nil {
		return 0, err
	}
	if port < 1 || port > math.MaxUint16 {
		return 0, errors.New("outside range")
	}
	return uint16(port), nil
}

func makeDefaultDialer() *net.Dialer {
	return &net.Dialer{KeepAlive: 5 * time.Minute}
}

func makeDefaultResolver() *net.Resolver {
	return net.DefaultResolver
}

func parseConnectTimeoutSetting(s string) (time.Duration, error) {
	timeout, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, err
	}
	if timeout < 0 {
		return 0, errors.New("negative timeout")
	}
	return time.Duration(timeout) * time.Second, nil
}

func makeConnectTimeoutDialFunc(timeout time.Duration) DialFunc {
	d := makeDefaultDialer()
	d.Timeout = timeout
	return d.DialContext
}