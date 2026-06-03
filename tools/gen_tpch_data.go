package main

import (
	"fmt"
	"math/rand"
	"os"
	"time"
)

func main() {
	rand.Seed(time.Now().UnixNano())

	file, err := os.Create("resources/tpch_data_insert.sql")
	if err != nil {
		panic(err)
	}
	defer file.Close()

	// REGION table (5 regions)
	regions := []struct {
		key  int
		name string
	}{
		{0, "AFRICA"},
		{1, "AMERICA"},
		{2, "ASIA"},
		{3, "EUROPE"},
		{4, "MIDDLE EAST"},
	}

	for _, r := range regions {
		fmt.Fprintf(file, "INSERT INTO region VALUES (%d, '%s', 'Region %s comment text');\n",
			r.key, r.name, r.name)
	}

	// NATION table (25 nations)
	nations := []struct {
		key      int
		name     string
		region   int
	}{
		{0, "ALGERIA", 0}, {1, "ARGENTINA", 1}, {2, "BRAZIL", 1},
		{3, "CANADA", 1}, {4, "EGYPT", 4}, {5, "ETHIOPIA", 0},
		{6, "FRANCE", 3}, {7, "GERMANY", 3}, {8, "INDIA", 2},
		{9, "INDONESIA", 2}, {10, "IRAN", 4}, {11, "IRAQ", 4},
		{12, "JAPAN", 2}, {13, "JORDAN", 4}, {14, "KENYA", 0},
		{15, "MOROCCO", 0}, {16, "MOZAMBIQUE", 0}, {17, "PERU", 1},
		{18, "CHINA", 2}, {19, "ROMANIA", 3}, {20, "SAUDI ARABIA", 4},
		{21, "VIETNAM", 2}, {22, "RUSSIA", 3}, {23, "UNITED KINGDOM", 3},
		{24, "UNITED STATES", 1},
	}

	for _, n := range nations {
		fmt.Fprintf(file, "INSERT INTO nation VALUES (%d, '%s', %d, 'Nation %s comment');\n",
			n.key, n.name, n.region, n.name)
	}

	// PART table (200 parts)
	fmt.Fprintf(file, "\n-- PART table (200 parts)\n")
	brands := []string{"Brand#11", "Brand#12", "Brand#13", "Brand#21", "Brand#22", "Brand#23"}
	types := []string{"STANDARD POLISHED TIN", "STANDARD BURNISHED BRASS", "PROMO BURNISHED COPPER",
		"SMALL PLATED STEEL", "MEDIUM ANODIZED BRASS", "LARGE BRUSHED COPPER"}
	containers := []string{"SM CASE", "SM BOX", "SM BAG", "MED BOX", "MED PKG", "LG CASE", "LG BOX"}

	for i := 1; i <= 200; i++ {
		brand := brands[rand.Intn(len(brands))]
		typ := types[rand.Intn(len(types))]
		container := containers[rand.Intn(len(containers))]
		size := rand.Intn(50) + 1
		price := float64(rand.Intn(200000)) / 100.0 + 10.0

		fmt.Fprintf(file, "INSERT INTO part VALUES (%d, 'Part_%d_name', 'Manufacturer#%d', '%s', '%s', %d, '%s', %.2f, 'Part %d comment text');\n",
			i, i, rand.Intn(5)+1, brand, typ, size, container, price, i)
	}

	// SUPPLIER table (100 suppliers)
	fmt.Fprintf(file, "\n-- SUPPLIER table (100 suppliers)\n")
	for i := 1; i <= 100; i++ {
		nationKey := rand.Intn(25)
		acctbal := float64(rand.Intn(200000)) / 100.0 - 999.99
		fmt.Fprintf(file, "INSERT INTO supplier VALUES (%d, 'Supplier#%09d', 'Address_%d', %d, '%02d-%03d-%03d-%04d', %.2f, 'Supplier %d comment');\n",
			i, i, i, nationKey, 10+rand.Intn(25), rand.Intn(999), rand.Intn(999), rand.Intn(9999), acctbal, i)
	}

	// PARTSUPP table (800 rows - 4 suppliers per part)
	fmt.Fprintf(file, "\n-- PARTSUPP table (800 rows)\n")
	for partKey := 1; partKey <= 200; partKey++ {
		for j := 0; j < 4; j++ {
			suppKey := ((partKey + j*10) % 100) + 1
			availqty := rand.Intn(9999) + 1
			supplycost := float64(rand.Intn(100000)) / 100.0 + 1.0
			fmt.Fprintf(file, "INSERT INTO partsupp VALUES (%d, %d, %d, %.2f, 'Partsupp comment for part %d supp %d');\n",
				partKey, suppKey, availqty, supplycost, partKey, suppKey)
		}
	}

	// CUSTOMER table (150 customers)
	fmt.Fprintf(file, "\n-- CUSTOMER table (150 customers)\n")
	segments := []string{"AUTOMOBILE", "BUILDING", "FURNITURE", "MACHINERY", "HOUSEHOLD"}
	for i := 1; i <= 150; i++ {
		nationKey := rand.Intn(25)
		acctbal := float64(rand.Intn(200000)) / 100.0 - 999.99
		segment := segments[rand.Intn(len(segments))]
		fmt.Fprintf(file, "INSERT INTO customer VALUES (%d, 'Customer#%09d', 'Address_%d', %d, '%02d-%03d-%03d-%04d', %.2f, '%s', 'Customer %d comment');\n",
			i, i, i, nationKey, 10+rand.Intn(25), rand.Intn(999), rand.Intn(999), rand.Intn(9999), acctbal, segment, i)
	}

	// ORDERS table (1500 orders)
	fmt.Fprintf(file, "\n-- ORDERS table (1500 orders)\n")
	priorities := []string{"1-URGENT", "2-HIGH", "3-MEDIUM", "4-NOT SPECIFIED", "5-LOW"}
	statuses := []string{"F", "O", "P"}

	for i := 1; i <= 1500; i++ {
		custKey := rand.Intn(150) + 1
		totalprice := float64(rand.Intn(50000000)) / 100.0 + 1000.0

		// Generate date between 1992-01-01 and 1998-12-31
		daysOffset := rand.Intn(2557) // ~7 years
		year := 1992 + daysOffset/365
		month := (daysOffset%365)/30 + 1
		day := (daysOffset%365)%30 + 1
		if day > 28 {
			day = 28
		}
		if month > 12 {
			month = 12
		}

		orderdate := fmt.Sprintf("%04d-%02d-%02d", year, month, day)
		priority := priorities[rand.Intn(len(priorities))]
		status := statuses[rand.Intn(len(statuses))]
		clerk := fmt.Sprintf("Clerk#%09d", rand.Intn(1000)+1)
		shippriority := 0

		fmt.Fprintf(file, "INSERT INTO orders VALUES (%d, %d, '%s', %.2f, '%s', '%s', '%s', %d, 'Order %d comment');\n",
			i, custKey, status, totalprice, orderdate, priority, clerk, shippriority, i)
	}

	// LINEITEM table (6000 lineitems - 4 per order)
	fmt.Fprintf(file, "\n-- LINEITEM table (6000 lineitems)\n")
	returnflags := []string{"N", "R", "A"}
	linestatuses := []string{"O", "F"}
	instructions := []string{"DELIVER IN PERSON", "COLLECT COD", "NONE", "TAKE BACK RETURN"}
	modes := []string{"REG AIR", "AIR", "RAIL", "SHIP", "TRUCK", "MAIL", "FOB"}

	for orderKey := 1; orderKey <= 1500; orderKey++ {
		numLines := rand.Intn(4) + 3 // 3-6 lines per order
		for lineNum := 1; lineNum <= numLines; lineNum++ {
			partKey := rand.Intn(200) + 1
			suppKey := ((partKey + (orderKey%10)) % 100) + 1
			quantity := rand.Intn(50) + 1
			extendedprice := float64(quantity) * (float64(rand.Intn(200000))/100.0 + 10.0)
			discount := float64(rand.Intn(11)) / 100.0
			tax := float64(rand.Intn(9)) / 100.0
			returnflag := returnflags[rand.Intn(len(returnflags))]
			linestatus := linestatuses[rand.Intn(len(linestatuses))]

			// Generate dates
			shipDays := rand.Intn(120)
			commitDays := rand.Intn(90) + 30
			receiptDays := shipDays + rand.Intn(30)

			shipdate := fmt.Sprintf("1994-%02d-%02d", (shipDays/30)%12+1, shipDays%28+1)
			commitdate := fmt.Sprintf("1994-%02d-%02d", (commitDays/30)%12+1, commitDays%28+1)
			receiptdate := fmt.Sprintf("1994-%02d-%02d", (receiptDays/30)%12+1, receiptDays%28+1)

			instruction := instructions[rand.Intn(len(instructions))]
			mode := modes[rand.Intn(len(modes))]

			fmt.Fprintf(file, "INSERT INTO lineitem VALUES (%d, %d, %d, %d, %d, %.2f, %.2f, %.2f, '%s', '%s', '%s', '%s', '%s', '%s', '%s', 'Lineitem comment');\n",
				orderKey, partKey, suppKey, lineNum, quantity, extendedprice, discount, tax,
				returnflag, linestatus, shipdate, commitdate, receiptdate, instruction, mode)
		}
	}

	fmt.Println("TPC-H test data generated: resources/tpch_data_insert.sql")
}
