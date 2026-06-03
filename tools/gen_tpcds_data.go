package main

// TPC-DS 数据生成器
// 支持使用 Docker 运行官方 dsdgen 或生成简化版数据
// Scale Factor 1 (1SF) ≈ 1GB 数据

import (
	"bufio"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

var r *rand.Rand

func init() {
	r = rand.New(rand.NewSource(time.Now().UnixNano()))
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: gen_tpcds_data <scale_factor> [output_dir]")
		fmt.Println("  scale_factor: 0.01 (10MB), 0.1 (100MB), 1 (1GB)")
		fmt.Println("  output_dir: default is resources/tpcds_data/")
		os.Exit(1)
	}

	sf := os.Args[1]
	outputDir := "resources/tpcds_data"
	if len(os.Args) > 2 {
		outputDir = os.Args[2]
	}

	// 创建输出目录
	os.MkdirAll(outputDir, 0755)

	// 尝试使用 Docker 生成数据
	if tryDockerDSGen(sf, outputDir) {
		fmt.Println("✓ TPC-DS data generated using Docker dsdgen")
		return
	}

	// Docker 不可用，使用简化版生成器
	fmt.Println("Docker not available, using simplified generator...")
	generateSimplifiedData(sf, outputDir)
}

// tryDockerDSGen: 尝试使用 Docker 运行官方 dsdgen
func tryDockerDSGen(sf, outputDir string) bool {
	// 检查 Docker 是否可用
	cmd := exec.Command("docker", "--version")
	if err := cmd.Run(); err != nil {
		fmt.Println("Docker not available:", err)
		return false
	}

	// 使用 Docker 镜像生成数据
	// 镜像: ghcr.io/scalytics/tpc-ds-docker:latest
	absPath, _ := filepath.Abs(outputDir)
	dockerCmd := fmt.Sprintf(
		"docker run --rm -v %s:/data ghcr.io/scalytics/tpc-ds-docker:latest ./dsdgen -scale %s -dir /data",
		absPath, sf)

	fmt.Println("Running:", dockerCmd)
	cmd = exec.Command("bash", "-c", dockerCmd)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		fmt.Println("Docker dsdgen failed:", err)
		return false
	}

	// 生成 LOAD DATA 脚本
	generateLoadScript(outputDir)
	return true
}

// generateSimplifiedData: 生成简化版 TPC-DS 数据
func generateSimplifiedData(sf, outputDir string) {
	scaleFactor := parseScaleFactorFloat(sf)

	fmt.Printf("Generating simplified TPC-DS data (SF=%s, scale=%.2f)...\n", sf, scaleFactor)

	// 计算各表行数 (基于 1SF 标准)
	counts := map[string]int{
		"date_dim":             73049,  // 200 years (固定)
		"time_dim":             86400,  // 24 hours (固定)
		"item":                 int(18000 * scaleFactor),
		"customer":             int(12000000 * scaleFactor),
		"customer_demographics": 1920800, // 固定
		"customer_address":     int(6000000 * scaleFactor),
		"household_demographics": 7200, // 固定
		"income_band":          20, // 固定
		"store":                int(12 * scaleFactor),
		"call_center":          int(6 * scaleFactor),
		"catalog_page":         int(11718 * scaleFactor),
		"web_site":             int(30 * scaleFactor),
		"web_page":             int(60 * scaleFactor),
		"warehouse":            int(5 * scaleFactor),
		"ship_mode":            20, // 固定
		"promotion":            int(300 * scaleFactor),
		"reason":               35, // 固定
		"inventory":            int(11745000 * scaleFactor),
		"store_sales":          int(2750473 * scaleFactor),
		"store_returns":        int(287514 * scaleFactor),
		"catalog_sales":        int(1441548 * scaleFactor),
		"catalog_returns":      int(144067 * scaleFactor),
		"web_sales":            int(719384 * scaleFactor),
		"web_returns":          int(71763 * scaleFactor),
	}

	// 生成各表数据
	generateDateDim(outputDir, counts["date_dim"])
	generateTimeDim(outputDir, counts["time_dim"])
	generateItem(outputDir, counts["item"])
	generateCustomer(outputDir, counts["customer"])
	generateCustomerDemographics(outputDir, counts["customer_demographics"])
	generateCustomerAddress(outputDir, counts["customer_address"])
	generateHouseholdDemographics(outputDir, counts["household_demographics"])
	generateIncomeBand(outputDir, counts["income_band"])
	generateStore(outputDir, counts["store"])
	generateCallCenter(outputDir, counts["call_center"])
	generateCatalogPage(outputDir, counts["catalog_page"])
	generateWebSite(outputDir, counts["web_site"])
	generateWebPage(outputDir, counts["web_page"])
	generateWarehouse(outputDir, counts["warehouse"])
	generateShipMode(outputDir, counts["ship_mode"])
	generatePromotion(outputDir, counts["promotion"])
	generateReason(outputDir, counts["reason"])
	generateInventory(outputDir, counts["inventory"])
	generateStoreSales(outputDir, counts["store_sales"])
	generateStoreReturns(outputDir, counts["store_returns"])
	generateCatalogSales(outputDir, counts["catalog_sales"])
	generateCatalogReturns(outputDir, counts["catalog_returns"])
	generateWebSales(outputDir, counts["web_sales"])
	generateWebReturns(outputDir, counts["web_returns"])

	// 生成 LOAD DATA 脚本
	generateLoadScript(outputDir)

	fmt.Println("\n✓ TPC-DS simplified data generated in:", outputDir)
	fmt.Println("To load data into MySQL:")
	fmt.Println("  mysql -h127.0.0.1 -P3306 -utpcc -pTaurus@123 tpcds < resources/tpcds_ddl.sql")
	fmt.Println("  mysql -h127.0.0.1 -P3306 -utpcc -pTaurus@123 tpcds < resources/tpcds_data/load_data.sql")
}

func parseScaleFactor(sf string) int {
	switch sf {
	case "0.01":
		return 10
	case "0.1":
		return 100
	case "1":
		return 1000
	default:
		fmt.Printf("Unknown scale factor: %s, using 1\n", sf)
		return 1000
	}
}

func parseScaleFactorFloat(sf string) float64 {
	switch sf {
	case "0.01":
		return 0.01
	case "0.1":
		return 0.1
	case "1":
		return 1.0
	case "10":
		return 10.0
	default:
		fmt.Printf("Unknown scale factor: %s, using 1.0\n", sf)
		return 1.0
	}
}

// ============================================================
// 维度表生成函数
// ============================================================

func generateDateDim(outputDir string, count int) {
	fmt.Printf("Generating date_dim (%d rows)...\n", count)
	f, _ := os.Create(filepath.Join(outputDir, "date_dim.dat"))
	defer f.Close()
	w := bufio.NewWriter(f)

	baseDate := time.Date(1900, 1, 1, 0, 0, 0, 0, time.UTC)
	for i := 0; i < count; i++ {
		d := baseDate.AddDate(0, 0, i)
		dateSK := 2415021 + i // Julian date
		dateID := fmt.Sprintf("%s%02d%02d", fmt.Sprintf("%d", d.Year()), d.Month(), d.Day())
		year := d.Year()
		month := int(d.Month())
		day := d.Day()
		dow := int(d.Weekday())
		dom := day
		moy := month
		qoy := (month-1)/3 + 1
		weekSeq := i / 7
		monthSeq := (year-1900)*12 + month - 1
		quarterSeq := (year-1900)*4 + (month-1)/3
		dayName := d.Weekday().String()
		quarterName := fmt.Sprintf("Q%d", qoy)
		holiday := "N"
		weekend := "N"
		if dow == 0 || dow == 6 {
			weekend = "Y"
		}

		fmt.Fprintf(w, "%d|%s|%s|%d|%d|%d|%d|%d|%d|%d|%d|%d|%d|%d|%s|%s|%s|%s|%s|%d|%d|%d|%d|N|N|N|N|N|\n",
			dateSK, dateID, d.Format("2006-01-02"),
			monthSeq, weekSeq, quarterSeq, year, dow, moy, dom, qoy,
			year, quarterSeq, weekSeq,
			dayName, quarterName, holiday, weekend, "N",
			0, 0, 0, 0)
	}
	w.Flush()
}

func generateTimeDim(outputDir string, count int) {
	fmt.Printf("Generating time_dim (%d rows)...\n", count)
	f, _ := os.Create(filepath.Join(outputDir, "time_dim.dat"))
	defer f.Close()
	w := bufio.NewWriter(f)

	for i := 0; i < count; i++ {
		hour := i / 3600
		minute := (i % 3600) / 60
		second := i % 60
		amPM := "AM"
		if hour >= 12 {
			amPM = "PM"
		}
		shift := "third"
		if hour >= 8 && hour < 16 {
			shift = "first"
		} else if hour >= 16 && hour < 24 {
			shift = "second"
		}

		fmt.Fprintf(w, "%d|%010d|%d|%d|%d|%d|%s|%s|third|unknown|\n",
			i, i, i, hour, minute, second, amPM, shift)
	}
	w.Flush()
}

func generateItem(outputDir string, count int) {
	fmt.Printf("Generating item (%d rows)...\n", count)
	f, _ := os.Create(filepath.Join(outputDir, "item.dat"))
	defer f.Close()
	w := bufio.NewWriter(f)

	categories := []string{"Books", "Music", "Electronics", "Sports", "Jewelry", "Home", "Children", "Shoes", "Women", "Men"}
	classes := []string{"science", "classical", "portable", "football", "rings", "furniture", "toddlers", "athletic", "dresses", "shirts"}
	brands := []string{"Brand#11", "Brand#12", "Brand#13", "Brand#21", "Brand#22"}

	for i := 1; i <= count; i++ {
		cat := categories[r.Intn(len(categories))]
		cls := classes[r.Intn(len(classes))]
		brand := brands[r.Intn(len(brands))]
		price := float64(r.Intn(100000)) / 100.0 + 1.0

		fmt.Fprintf(w, "%d|AAAAAAAAAAAA%04d||%s|%.2f|%.2f|%d|%s|%d|%s|%d|%s|%d|%s|%s|%s|%s|%s|%s|%s|\n",
			i, i,
			fmt.Sprintf("Item description %d with some details", i),
			price, price*0.8,
			r.Intn(1000)+1, brand,
			r.Intn(100)+1, cls,
			r.Intn(10)+1, cat,
			r.Intn(1000)+1, fmt.Sprintf("Manufact#%d", r.Intn(10)+1),
			"Medium", "01oz 02oz 03oz", "Unknown", "Each", "N/A",
			fmt.Sprintf("Product#%d", i))
	}
	w.Flush()
}

func generateCustomer(outputDir string, count int) {
	fmt.Printf("Generating customer (%d rows)...\n", count)
	f, _ := os.Create(filepath.Join(outputDir, "customer.dat"))
	defer f.Close()
	w := bufio.NewWriter(f)

	salutations := []string{"Mr.", "Mrs.", "Ms.", "Dr.", "Sir", "Madam"}
	countries := []string{"United States", "Canada", "Mexico", "UK", "France", "Germany", "Japan", "China", "India", "Brazil"}

	for i := 1; i <= count; i++ {
		sal := salutations[r.Intn(len(salutations))]
		fname := fmt.Sprintf("FirstName%05d", r.Intn(10000))
		lname := fmt.Sprintf("LastName%05d", r.Intn(10000))
		country := countries[r.Intn(len(countries))]
		email := fmt.Sprintf("%s.%s@email.com", strings.ToLower(fname), strings.ToLower(lname))

		fmt.Fprintf(w, "%d|AAAAAAAAAAAA%04d|%d|%d|%d|%d|%d|%s|%s|%s|Y|%d|%d|%d|%s||%s||\n",
			i, i,
			r.Intn(1920800)+1, // cdemo_sk
			r.Intn(7200)+1,    // hdemo_sk
			r.Intn(6000000)+1, // addr_sk
			r.Intn(73049)+1,   // first_shipto_date_sk
			r.Intn(73049)+1,   // first_sales_date_sk
			sal, fname, lname,
			r.Intn(31)+1, r.Intn(12)+1, 1950+r.Intn(50),
			country, email)
	}
	w.Flush()
}

func generateCustomerDemographics(outputDir string, count int) {
	fmt.Printf("Generating customer_demographics (%d rows)...\n", count)
	f, _ := os.Create(filepath.Join(outputDir, "customer_demographics.dat"))
	defer f.Close()
	w := bufio.NewWriter(f)

	genders := []string{"M", "F", "U"}
	maritalStatus := []string{"M", "S", "W", "D", "U"}
	education := []string{"Primary", "Secondary", "College", "Bachelor", "Master", "PhD", "Unknown"}
	creditRatings := []string{"Excellent", "Good", "Fair", "Poor", "Unknown"}

	for i := 1; i <= count; i++ {
		fmt.Fprintf(w, "%d|%s|%s|%s|%d|%s|%d|%d|%d|\n",
			i,
			genders[r.Intn(len(genders))],
			maritalStatus[r.Intn(len(maritalStatus))],
			education[r.Intn(len(education))],
			r.Intn(10000)+1000,
			creditRatings[r.Intn(len(creditRatings))],
			r.Intn(6),
			r.Intn(4),
			r.Intn(4))
	}
	w.Flush()
}

func generateCustomerAddress(outputDir string, count int) {
	fmt.Printf("Generating customer_address (%d rows)...\n", count)
	f, _ := os.Create(filepath.Join(outputDir, "customer_address.dat"))
	defer f.Close()
	w := bufio.NewWriter(f)

	states := []string{"CA", "NY", "TX", "FL", "IL", "PA", "OH", "GA", "NC", "MI"}
	countries := []string{"United States", "Canada"}
	locationTypes := []string{"single family", "apartment", "condo", "unknown"}

	for i := 1; i <= count; i++ {
		state := states[r.Intn(len(states))]
		city := fmt.Sprintf("City%04d", r.Intn(1000))
		zip := fmt.Sprintf("%05d", r.Intn(99999))

		fmt.Fprintf(w, "%d|AAAAAAAAAAAA%04d|%d|%s Street|%s|%s|%s|%s|%s|%s|%s|%.2f|%s|\n",
			i, i,
			r.Intn(999)+1,
			fmt.Sprintf("Street %d", r.Intn(1000)),
			"St", "Suite",
			city, "County", state, zip,
			countries[r.Intn(len(countries))],
			-5.0+float64(r.Intn(1000))/100.0,
			locationTypes[r.Intn(len(locationTypes))])
	}
	w.Flush()
}

func generateHouseholdDemographics(outputDir string, count int) {
	fmt.Printf("Generating household_demographics (%d rows)...\n", count)
	f, _ := os.Create(filepath.Join(outputDir, "household_demographics.dat"))
	defer f.Close()
	w := bufio.NewWriter(f)

	buyPotentials := []string{"0-500", "501-1000", "1001-5000", "5001-10000", "10001+"}

	for i := 1; i <= count; i++ {
		fmt.Fprintf(w, "%d|%d|%s|%d|%d|\n",
			i,
			r.Intn(20)+1,
			buyPotentials[r.Intn(len(buyPotentials))],
			r.Intn(6),
			r.Intn(5))
	}
	w.Flush()
}

func generateIncomeBand(outputDir string, count int) {
	fmt.Printf("Generating income_band (%d rows)...\n", count)
	f, _ := os.Create(filepath.Join(outputDir, "income_band.dat"))
	defer f.Close()
	w := bufio.NewWriter(f)

	for i := 1; i <= count; i++ {
		lower := (i - 1) * 10000
		upper := i*10000 - 1
		fmt.Fprintf(w, "%d|%d|%d|\n", i, lower, upper)
	}
	w.Flush()
}

func generateStore(outputDir string, count int) {
	fmt.Printf("Generating store (%d rows)...\n", count)
	f, _ := os.Create(filepath.Join(outputDir, "store.dat"))
	defer f.Close()
	w := bufio.NewWriter(f)

	for i := 1; i <= count; i++ {
		fmt.Fprintf(w, "%d|AAAAAAAAAAAA%04d|||N|Store#%d|%d|%d|8AM-10PM|Manager%d|%d|Medium|Store %d description|Manager%d|%d|Division%d|%d|Company%d|%d|Main St|St|Suite|City%d|County|CA|%05d|United States|%.2f|%.2f|\n",
			i, i, i,
			r.Intn(500)+10,
			r.Intn(10000)+1000,
			r.Intn(100)+1,
			r.Intn(10)+1,
			r.Intn(10)+1, i,
			r.Intn(10)+1,
			r.Intn(1000),
			r.Intn(99999),
			-8.0, 0.05)
	}
	w.Flush()
}

func generateCallCenter(outputDir string, count int) {
	fmt.Printf("Generating call_center (%d rows)...\n", count)
	f, _ := os.Create(filepath.Join(outputDir, "call_center.dat"))
	defer f.Close()
	w := bufio.NewWriter(f)

	for i := 1; i <= count; i++ {
		fmt.Fprintf(w, "%d|AAAAAAAAAAAA%04d|||N|N|CallCenter#%d|large|%d|%d|24 hours|Manager%d|%d|Medium|Call center description|Manager%d|%d|Division%d|%d|Company%d|123|Main St|St|Suite|City%d|County|CA|%05d|United States|%.2f|%.2f|\n",
			i, i, i,
			r.Intn(100)+10,
			r.Intn(10000)+1000,
			r.Intn(100)+1,
			r.Intn(10)+1,
			r.Intn(10)+1, i,
			r.Intn(1000),
			r.Intn(99999),
			-8.0, 0.05)
	}
	w.Flush()
}

func generateCatalogPage(outputDir string, count int) {
	fmt.Printf("Generating catalog_page (%d rows)...\n", count)
	f, _ := os.Create(filepath.Join(outputDir, "catalog_page.dat"))
	defer f.Close()
	w := bufio.NewWriter(f)

	types := []string{"cover", "index", "ad", "product"}

	for i := 1; i <= count; i++ {
		fmt.Fprintf(w, "%d|AAAAAAAAAAAA%04d|%d|%d|Department%d|%d|%d|Catalog page %d description|%s|\n",
			i, i,
			r.Intn(73049)+1,
			r.Intn(73049)+1,
			r.Intn(10)+1,
			r.Intn(100)+1,
			r.Intn(100)+1, i,
			types[r.Intn(len(types))])
	}
	w.Flush()
}

func generateWebSite(outputDir string, count int) {
	fmt.Printf("Generating web_site (%d rows)...\n", count)
	f, _ := os.Create(filepath.Join(outputDir, "web_site.dat"))
	defer f.Close()
	w := bufio.NewWriter(f)

	for i := 1; i <= count; i++ {
		fmt.Fprintf(w, "%d|AAAAAAAAAAAA%04d|||N|N|WebSite#%d|medium|Manager%d|%d|Medium|Web site description|Manager%d|%d|Company%d|123|Main St|St|Suite|City%d|County|CA|%05d|United States|%.2f|%.2f|\n",
			i, i, i,
			r.Intn(100)+1,
			r.Intn(10)+1,
			r.Intn(10)+1, i,
			r.Intn(1000),
			r.Intn(99999),
			-8.0, 0.05)
	}
	w.Flush()
}

func generateWebPage(outputDir string, count int) {
	fmt.Printf("Generating web_page (%d rows)...\n", count)
	f, _ := os.Create(filepath.Join(outputDir, "web_page.dat"))
	defer f.Close()
	w := bufio.NewWriter(f)

	types := []string{"home", "product", "checkout", "cart", "review"}

	for i := 1; i <= count; i++ {
		fmt.Fprintf(w, "%d|AAAAAAAAAAAA%04d|||N|N|Y|%d|http://www.example.com/page%d|%s|%d|%d|%d|%d|\n",
			i, i,
			r.Intn(12000000)+1,
			r.Intn(1000),
			types[r.Intn(len(types))],
			r.Intn(10000)+100,
			r.Intn(100)+1,
			r.Intn(50)+1,
			r.Intn(10)+1)
	}
	w.Flush()
}

func generateWarehouse(outputDir string, count int) {
	fmt.Printf("Generating warehouse (%d rows)...\n", count)
	f, _ := os.Create(filepath.Join(outputDir, "warehouse.dat"))
	defer f.Close()
	w := bufio.NewWriter(f)

	for i := 1; i <= count; i++ {
		fmt.Fprintf(w, "%d|AAAAAAAAAAAA%04d|Warehouse#%d|%d|123|Main St|St|Suite|City%d|County|CA|%05d|United States|%.2f|\n",
			i, i, i,
			r.Intn(1000000)+100000,
			r.Intn(1000),
			r.Intn(99999),
			-8.0)
	}
	w.Flush()
}

func generateShipMode(outputDir string, count int) {
	fmt.Printf("Generating ship_mode (%d rows)...\n", count)
	f, _ := os.Create(filepath.Join(outputDir, "ship_mode.dat"))
	defer f.Close()
	w := bufio.NewWriter(f)

	types := []string{"EXPRESS", "REGULAR", "OVERNIGHT", "LIBRARY", "NEXT DAY"}
	codes := []string{"AIR", "GROUND", "SEA", "RAIL", "UNKNOWN"}
	carriers := []string{"FedEx", "UPS", "USPS", "DHL", "Freight"}

	for i := 1; i <= count; i++ {
		fmt.Fprintf(w, "%d|AAAAAAAAAAAA%04d|%s|%s|%s|Contract%d|\n",
			i, i,
			types[r.Intn(len(types))],
			codes[r.Intn(len(codes))],
			carriers[r.Intn(len(carriers))],
			r.Intn(100)+1)
	}
	w.Flush()
}

func generatePromotion(outputDir string, count int) {
	fmt.Printf("Generating promotion (%d rows)...\n", count)
	f, _ := os.Create(filepath.Join(outputDir, "promotion.dat"))
	defer f.Close()
	w := bufio.NewWriter(f)

	for i := 1; i <= count; i++ {
		fmt.Fprintf(w, "%d|AAAAAAAAAAAA%04d|%d|%d|%d|%.2f|%d|Promo%d|Y|N|N|N|N|N|N|N|Promo %d details|Unknown|N|\n",
			i, i,
			r.Intn(73049)+1,
			r.Intn(73049)+1,
			r.Intn(18000)+1,
			float64(r.Intn(10000))/100.0,
			r.Intn(100)+1, i, i)
	}
	w.Flush()
}

func generateReason(outputDir string, count int) {
	fmt.Printf("Generating reason (%d rows)...\n", count)
	f, _ := os.Create(filepath.Join(outputDir, "reason.dat"))
	defer f.Close()
	w := bufio.NewWriter(f)

	reasons := []string{
		"Defective product", "Wrong item", "Gift exchange", "Changed mind",
		"Found better price", "Duplicate purchase", "Not as described",
		"Damaged in shipping", "Arrived too late", "Quality issues",
		"Wrong size", "Missing parts", "Incompatible", "Better alternative",
		"No longer needed", "Budget constraints", "Gift recipient didn't like",
		"Product expired", "Recalled product", "Customer service issue",
		"Website error", "Pricing error", "Shipping delay", "Package lost",
		"Customs issues", "Weather delay", "Inventory shortage", "Backorder cancellation",
		"Subscription cancelled", "Trial period ended", "Warranty claim",
		"Product recall", "Manufacturer defect", "Safety concern", "Regulatory compliance"}

	for i := 1; i <= count; i++ {
		desc := reasons[i%len(reasons)]
		fmt.Fprintf(w, "%d|AAAAAAAAAAAA%04d|%s|\n", i, i, desc)
	}
	w.Flush()
}

// ============================================================
// 事实表生成函数
// ============================================================

func generateInventory(outputDir string, count int) {
	fmt.Printf("Generating inventory (%d rows)...\n", count)
	f, _ := os.Create(filepath.Join(outputDir, "inventory.dat"))
	defer f.Close()
	w := bufio.NewWriter(f)

	for i := 0; i < count; i++ {
		dateSK := 2450816 + r.Intn(730) // 2 years
		itemSK := r.Intn(18000) + 1
		warehouseSK := r.Intn(5) + 1
		qty := r.Intn(1000)

		fmt.Fprintf(w, "%d|%d|%d|%d|\n", dateSK, itemSK, warehouseSK, qty)
	}
	w.Flush()
}

func generateStoreSales(outputDir string, count int) {
	fmt.Printf("Generating store_sales (%d rows)...\n", count)
	f, _ := os.Create(filepath.Join(outputDir, "store_sales.dat"))
	defer f.Close()
	w := bufio.NewWriter(f)

	for i := 0; i < count; i++ {
		dateSK := 2450816 + r.Intn(730)
		timeSK := r.Intn(86400)
		itemSK := r.Intn(18000) + 1
		customerSK := r.Intn(12000000) + 1
		cdemoSK := r.Intn(1920800) + 1
		hdemoSK := r.Intn(7200) + 1
		addrSK := r.Intn(6000000) + 1
		storeSK := r.Intn(12) + 1
		promoSK := r.Intn(300) + 1
		ticketNum := i/4 + 1
		qty := r.Intn(100) + 1
		wholesale := float64(r.Intn(10000)) / 100.0
		listPrice := wholesale * (1.0 + float64(r.Intn(50))/100.0)
		salesPrice := listPrice * (1.0 - float64(r.Intn(30))/100.0)
		discount := listPrice - salesPrice
		extDiscount := discount * float64(qty)
		extSales := salesPrice * float64(qty)
		extWholesale := wholesale * float64(qty)
		extList := listPrice * float64(qty)
		tax := extSales * 0.08
		coupon := 0.0
		if r.Intn(10) == 0 {
			coupon = extSales * 0.1
		}
		netPaid := extSales - coupon
		netPaidTax := netPaid + tax
		netProfit := netPaid - extWholesale

		fmt.Fprintf(w, "%d|%d|%d|%d|%d|%d|%d|%d|%d|%d|%d|%.2f|%.2f|%.2f|%.2f|%.2f|%.2f|%.2f|%.2f|%.2f|%.2f|%.2f|%.2f|\n",
			dateSK, timeSK, itemSK, customerSK, cdemoSK, hdemoSK, addrSK, storeSK, promoSK, ticketNum, qty,
			wholesale, listPrice, salesPrice, extDiscount, extSales, extWholesale, extList, tax, coupon, netPaid, netPaidTax, netProfit)
	}
	w.Flush()
}

func generateStoreReturns(outputDir string, count int) {
	fmt.Printf("Generating store_returns (%d rows)...\n", count)
	f, _ := os.Create(filepath.Join(outputDir, "store_returns.dat"))
	defer f.Close()
	w := bufio.NewWriter(f)

	for i := 0; i < count; i++ {
		dateSK := 2450816 + r.Intn(730)
		timeSK := r.Intn(86400)
		itemSK := r.Intn(18000) + 1
		customerSK := r.Intn(12000000) + 1
		cdemoSK := r.Intn(1920800) + 1
		hdemoSK := r.Intn(7200) + 1
		addrSK := r.Intn(6000000) + 1
		storeSK := r.Intn(12) + 1
		reasonSK := r.Intn(35) + 1
		ticketNum := i/2 + 1
		qty := r.Intn(10) + 1
		returnAmt := float64(r.Intn(10000)) / 100.0
		returnTax := returnAmt * 0.08
		returnAmtTax := returnAmt + returnTax
		fee := 10.0
		returnShipCost := returnAmt * 0.1
		refundedCash := returnAmt * 0.8
		reversedCharge := returnAmt * 0.1
		storeCredit := returnAmt * 0.1
		netLoss := returnAmt - refundedCash - reversedCharge - storeCredit + fee

		fmt.Fprintf(w, "%d|%d|%d|%d|%d|%d|%d|%d|%d|%d|%d|%.2f|%.2f|%.2f|%.2f|%.2f|%.2f|%.2f|%.2f|%.2f|\n",
			dateSK, timeSK, itemSK, customerSK, cdemoSK, hdemoSK, addrSK, storeSK, reasonSK, ticketNum, qty,
			returnAmt, returnTax, returnAmtTax, fee, returnShipCost, refundedCash, reversedCharge, storeCredit, netLoss)
	}
	w.Flush()
}

func generateCatalogSales(outputDir string, count int) {
	fmt.Printf("Generating catalog_sales (%d rows)...\n", count)
	f, _ := os.Create(filepath.Join(outputDir, "catalog_sales.dat"))
	defer f.Close()
	w := bufio.NewWriter(f)

	for i := 0; i < count; i++ {
		dateSK := 2450816 + r.Intn(730)
		timeSK := r.Intn(86400)
		shipDateSK := dateSK + r.Intn(30)
		billCustSK := r.Intn(12000000) + 1
		billCdemoSK := r.Intn(1920800) + 1
		billHdemoSK := r.Intn(7200) + 1
		billAddrSK := r.Intn(6000000) + 1
		shipCustSK := billCustSK
		shipCdemoSK := billCdemoSK
		shipHdemoSK := billHdemoSK
		shipAddrSK := billAddrSK
		callCenterSK := r.Intn(6) + 1
		catalogPageSK := r.Intn(11718) + 1
		shipModeSK := r.Intn(20) + 1
		warehouseSK := r.Intn(5) + 1
		itemSK := r.Intn(18000) + 1
		promoSK := r.Intn(300) + 1
		orderNum := i/3 + 1
		qty := r.Intn(50) + 1
		wholesale := float64(r.Intn(10000)) / 100.0
		listPrice := wholesale * (1.0 + float64(r.Intn(50))/100.0)
		salesPrice := listPrice * (1.0 - float64(r.Intn(30))/100.0)
		discount := listPrice - salesPrice
		extDiscount := discount * float64(qty)
		extSales := salesPrice * float64(qty)
		extWholesale := wholesale * float64(qty)
		extList := listPrice * float64(qty)
		tax := extSales * 0.08
		coupon := 0.0
		if r.Intn(10) == 0 {
			coupon = extSales * 0.1
		}
		extShipCost := extSales * 0.05
		netPaid := extSales - coupon
		netPaidTax := netPaid + tax
		netPaidShip := netPaid + extShipCost
		netPaidShipTax := netPaidShip + tax
		netProfit := netPaid - extWholesale

		fmt.Fprintf(w, "%d|%d|%d|%d|%d|%d|%d|%d|%d|%d|%d|%d|%d|%d|%d|%d|%d|%d|%.2f|%.2f|%.2f|%.2f|%.2f|%.2f|%.2f|%.2f|%.2f|%.2f|%.2f|%.2f|%.2f|%.2f|%.2f|\n",
			dateSK, timeSK, shipDateSK,
			billCustSK, billCdemoSK, billHdemoSK, billAddrSK,
			shipCustSK, shipCdemoSK, shipHdemoSK, shipAddrSK,
			callCenterSK, catalogPageSK, shipModeSK, warehouseSK,
			itemSK, promoSK, orderNum, qty,
			wholesale, listPrice, salesPrice, extDiscount, extSales, extWholesale, extList, tax, coupon, extShipCost,
			netPaid, netPaidTax, netPaidShip, netPaidShipTax, netProfit)
	}
	w.Flush()
}

func generateCatalogReturns(outputDir string, count int) {
	fmt.Printf("Generating catalog_returns (%d rows)...\n", count)
	f, _ := os.Create(filepath.Join(outputDir, "catalog_returns.dat"))
	defer f.Close()
	w := bufio.NewWriter(f)

	for i := 0; i < count; i++ {
		dateSK := 2450816 + r.Intn(730)
		timeSK := r.Intn(86400)
		itemSK := r.Intn(18000) + 1
		refundCustSK := r.Intn(12000000) + 1
		refundCdemoSK := r.Intn(1920800) + 1
		refundHdemoSK := r.Intn(7200) + 1
		refundAddrSK := r.Intn(6000000) + 1
		returnCustSK := refundCustSK
		returnCdemoSK := refundCdemoSK
		returnHdemoSK := refundHdemoSK
		returnAddrSK := refundAddrSK
		callCenterSK := r.Intn(6) + 1
		catalogPageSK := r.Intn(11718) + 1
		shipModeSK := r.Intn(20) + 1
		warehouseSK := r.Intn(5) + 1
		reasonSK := r.Intn(35) + 1
		orderNum := i/2 + 1
		qty := r.Intn(10) + 1
		returnAmt := float64(r.Intn(10000)) / 100.0
		returnTax := returnAmt * 0.08
		returnAmtTax := returnAmt + returnTax
		fee := 15.0
		returnShipCost := returnAmt * 0.1
		refundedCash := returnAmt * 0.8
		reversedCharge := returnAmt * 0.1
		storeCredit := returnAmt * 0.1
		netLoss := returnAmt - refundedCash - reversedCharge - storeCredit + fee

		fmt.Fprintf(w, "%d|%d|%d|%d|%d|%d|%d|%d|%d|%d|%d|%d|%d|%d|%d|%d|%d|%d|%.2f|%.2f|%.2f|%.2f|%.2f|%.2f|%.2f|%.2f|%.2f|\n",
			dateSK, timeSK, itemSK,
			refundCustSK, refundCdemoSK, refundHdemoSK, refundAddrSK,
			returnCustSK, returnCdemoSK, returnHdemoSK, returnAddrSK,
			callCenterSK, catalogPageSK, shipModeSK, warehouseSK, reasonSK, orderNum, qty,
			returnAmt, returnTax, returnAmtTax, fee, returnShipCost, refundedCash, reversedCharge, storeCredit, netLoss)
	}
	w.Flush()
}

func generateWebSales(outputDir string, count int) {
	fmt.Printf("Generating web_sales (%d rows)...\n", count)
	f, _ := os.Create(filepath.Join(outputDir, "web_sales.dat"))
	defer f.Close()
	w := bufio.NewWriter(f)

	for i := 0; i < count; i++ {
		dateSK := 2450816 + r.Intn(730)
		timeSK := r.Intn(86400)
		shipDateSK := dateSK + r.Intn(30)
		itemSK := r.Intn(18000) + 1
		billCustSK := r.Intn(12000000) + 1
		billCdemoSK := r.Intn(1920800) + 1
		billHdemoSK := r.Intn(7200) + 1
		billAddrSK := r.Intn(6000000) + 1
		shipCustSK := billCustSK
		shipCdemoSK := billCdemoSK
		shipHdemoSK := billHdemoSK
		shipAddrSK := billAddrSK
		webPageSK := r.Intn(60) + 1
		webSiteSK := r.Intn(30) + 1
		shipModeSK := r.Intn(20) + 1
		warehouseSK := r.Intn(5) + 1
		promoSK := r.Intn(300) + 1
		orderNum := i/3 + 1
		qty := r.Intn(50) + 1
		wholesale := float64(r.Intn(10000)) / 100.0
		listPrice := wholesale * (1.0 + float64(r.Intn(50))/100.0)
		salesPrice := listPrice * (1.0 - float64(r.Intn(30))/100.0)
		discount := listPrice - salesPrice
		extDiscount := discount * float64(qty)
		extSales := salesPrice * float64(qty)
		extWholesale := wholesale * float64(qty)
		extList := listPrice * float64(qty)
		tax := extSales * 0.08
		coupon := 0.0
		if r.Intn(10) == 0 {
			coupon = extSales * 0.1
		}
		extShipCost := extSales * 0.05
		netPaid := extSales - coupon
		netPaidTax := netPaid + tax
		netPaidShip := netPaid + extShipCost
		netPaidShipTax := netPaidShip + tax
		netProfit := netPaid - extWholesale

		fmt.Fprintf(w, "%d|%d|%d|%d|%d|%d|%d|%d|%d|%d|%d|%d|%d|%d|%d|%d|%d|%d|%.2f|%.2f|%.2f|%.2f|%.2f|%.2f|%.2f|%.2f|%.2f|%.2f|%.2f|%.2f|%.2f|%.2f|%.2f|\n",
			dateSK, timeSK, shipDateSK, itemSK,
			billCustSK, billCdemoSK, billHdemoSK, billAddrSK,
			shipCustSK, shipCdemoSK, shipHdemoSK, shipAddrSK,
			webPageSK, webSiteSK, shipModeSK, warehouseSK, promoSK, orderNum, qty,
			wholesale, listPrice, salesPrice, extDiscount, extSales, extWholesale, extList, tax, coupon, extShipCost,
			netPaid, netPaidTax, netPaidShip, netPaidShipTax, netProfit)
	}
	w.Flush()
}

func generateWebReturns(outputDir string, count int) {
	fmt.Printf("Generating web_returns (%d rows)...\n", count)
	f, _ := os.Create(filepath.Join(outputDir, "web_returns.dat"))
	defer f.Close()
	w := bufio.NewWriter(f)

	for i := 0; i < count; i++ {
		dateSK := 2450816 + r.Intn(730)
		timeSK := r.Intn(86400)
		itemSK := r.Intn(18000) + 1
		refundCustSK := r.Intn(12000000) + 1
		refundCdemoSK := r.Intn(1920800) + 1
		refundHdemoSK := r.Intn(7200) + 1
		refundAddrSK := r.Intn(6000000) + 1
		returnCustSK := refundCustSK
		returnCdemoSK := refundCdemoSK
		returnHdemoSK := refundHdemoSK
		returnAddrSK := refundAddrSK
		webPageSK := r.Intn(60) + 1
		reasonSK := r.Intn(35) + 1
		orderNum := i/2 + 1
		qty := r.Intn(10) + 1
		returnAmt := float64(r.Intn(10000)) / 100.0
		returnTax := returnAmt * 0.08
		returnAmtTax := returnAmt + returnTax
		fee := 12.0
		returnShipCost := returnAmt * 0.1
		refundedCash := returnAmt * 0.8
		reversedCharge := returnAmt * 0.1
		accountCredit := returnAmt * 0.1
		netLoss := returnAmt - refundedCash - reversedCharge - accountCredit + fee

		fmt.Fprintf(w, "%d|%d|%d|%d|%d|%d|%d|%d|%d|%d|%d|%d|%d|%d|%d|%.2f|%.2f|%.2f|%.2f|%.2f|%.2f|%.2f|%.2f|%.2f|\n",
			dateSK, timeSK, itemSK,
			refundCustSK, refundCdemoSK, refundHdemoSK, refundAddrSK,
			returnCustSK, returnCdemoSK, returnHdemoSK, returnAddrSK,
			webPageSK, reasonSK, orderNum, qty,
			returnAmt, returnTax, returnAmtTax, fee, returnShipCost, refundedCash, reversedCharge, accountCredit, netLoss)
	}
	w.Flush()
}

// generateLoadScript: 生成 LOAD DATA 脚本
func generateLoadScript(outputDir string) {
	fmt.Println("Generating load_data.sql...")
	f, _ := os.Create(filepath.Join(outputDir, "load_data.sql"))
	defer f.Close()
	w := bufio.NewWriter(f)

	fmt.Fprintln(w, "-- TPC-DS Data Loading Script")
	fmt.Fprintln(w, "-- Generated by gen_tpcds_data.go")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "SET FOREIGN_KEY_CHECKS = 0;")
	fmt.Fprintln(w, "SET UNIQUE_CHECKS = 0;")
	fmt.Fprintln(w, "SET SQL_MODE = '';")
	fmt.Fprintln(w, "")

	tables := []string{
		"date_dim", "time_dim", "item", "customer", "customer_demographics",
		"customer_address", "household_demographics", "income_band", "store",
		"call_center", "catalog_page", "web_site", "web_page", "warehouse",
		"ship_mode", "promotion", "reason", "inventory", "store_sales",
		"store_returns", "catalog_sales", "catalog_returns", "web_sales", "web_returns"}

	for _, table := range tables {
		fmt.Fprintf(w, "LOAD DATA LOCAL INFILE '%s.dat' INTO TABLE %s FIELDS TERMINATED BY '|' LINES TERMINATED BY '|\\n';\n",
			table, table)
	}

	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "SET FOREIGN_KEY_CHECKS = 1;")
	fmt.Fprintln(w, "SET UNIQUE_CHECKS = 1;")
	w.Flush()
}
