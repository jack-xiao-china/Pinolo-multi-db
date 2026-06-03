-- TPC-H 22 Standard Queries
-- TPC Benchmark H v3.0.1

-- Query 1: Pricing Summary Report
SELECT
    l_returnflag,
    l_linestatus,
    sum(l_quantity) as sum_qty,
    sum(l_extendedprice) as sum_base_price,
    sum(l_extendedprice * (1 - l_discount)) as sum_disc_price,
    sum(l_extendedprice * (1 - l_discount) * (1 + l_tax)) as sum_charge,
    avg(l_quantity) as avg_qty,
    avg(l_extendedprice) as avg_price,
    avg(l_discount) as avg_disc,
    count(*) as count_order
FROM lineitem
WHERE l_shipdate <= date '1998-12-01' - interval '90' day
GROUP BY l_returnflag, l_linestatus
ORDER BY l_returnflag, l_linestatus;

-- Query 2: Minimum Cost Supplier
SELECT
    s_acctbal,
    s_name,
    n_name,
    p_partkey,
    p_mfgr,
    s_address,
    s_phone,
    s_comment
FROM part, supplier, partsupp, nation, region
WHERE
    p_partkey = ps_partkey
    AND s_suppkey = ps_suppkey
    AND p_size = 15
    AND p_type LIKE '%BRASS'
    AND s_nationkey = n_nationkey
    AND n_regionkey = r_regionkey
    AND r_name = 'EUROPE'
    AND ps_supplycost = (
        SELECT min(ps_supplycost)
        FROM partsupp, supplier, nation, region
        WHERE
            p_partkey = ps_partkey
            AND s_suppkey = ps_suppkey
            AND s_nationkey = n_nationkey
            AND n_regionkey = r_regionkey
            AND r_name = 'EUROPE'
    )
ORDER BY s_acctbal desc, n_name, s_name, p_partkey
LIMIT 100;

-- Query 3: Shipping Priority
SELECT
    l_orderkey,
    sum(l_extendedprice * (1 - l_discount)) as revenue,
    o_orderdate,
    o_shippriority
FROM customer, orders, lineitem
WHERE
    c_mktsegment = 'BUILDING'
    AND c_custkey = o_custkey
    AND l_orderkey = o_orderkey
    AND o_orderdate < date '1995-03-15'
    AND l_shipdate > date '1995-03-15'
GROUP BY l_orderkey, o_orderdate, o_shippriority
ORDER BY revenue desc, o_orderdate
LIMIT 10;

-- Query 4: Order Priority Checking
SELECT
    o_orderpriority,
    count(*) as order_count
FROM orders
WHERE
    o_orderdate >= date '1993-07-01'
    AND o_orderdate < date '1993-07-01' + interval '3' month
    AND EXISTS (
        SELECT *
        FROM lineitem
        WHERE l_orderkey = o_orderkey
            AND l_commitdate < l_receiptdate
    )
GROUP BY o_orderpriority
ORDER BY o_orderpriority;

-- Query 5: Local Supplier Volume
SELECT
    n_name,
    sum(l_extendedprice * (1 - l_discount)) as revenue
FROM customer, orders, lineitem, supplier, nation, region
WHERE
    c_custkey = o_custkey
    AND l_orderkey = o_orderkey
    AND l_suppkey = s_suppkey
    AND c_nationkey = s_nationkey
    AND s_nationkey = n_nationkey
    AND n_regionkey = r_regionkey
    AND r_name = 'ASIA'
    AND o_orderdate >= date '1994-01-01'
    AND o_orderdate < date '1994-01-01' + interval '1' year
GROUP BY n_name
ORDER BY revenue desc;

-- Query 6: Forecasting Revenue Change
SELECT
    sum(l_extendedprice * l_discount) as revenue
FROM lineitem
WHERE
    l_shipdate >= date '1994-01-01'
    AND l_shipdate < date '1994-01-01' + interval '1' year
    AND l_discount between 0.06 - 0.01 AND 0.06 + 0.01
    AND l_quantity < 24;

-- Query 7: Volume Shipping
SELECT
    supp_nation,
    cust_nation,
    l_year,
    sum(volume) as revenue
FROM (
    SELECT
        n1.n_name as supp_nation,
        n2.n_name as cust_nation,
        extract(year from l_shipdate) as l_year,
        l_extendedprice * (1 - l_discount) as volume
    FROM supplier, lineitem, orders, customer, nation n1, nation n2
    WHERE
        s_suppkey = l_suppkey
        AND o_orderkey = l_orderkey
        AND c_custkey = o_custkey
        AND s_nationkey = n1.n_nationkey
        AND c_nationkey = n2.n_nationkey
        AND (
            (n1.n_name = 'FRANCE' AND n2.n_name = 'GERMANY')
            OR (n1.n_name = 'GERMANY' AND n2.n_name = 'FRANCE')
        )
        AND l_shipdate between date '1995-01-01' AND date '1996-12-31'
) as shipping
GROUP BY supp_nation, cust_nation, l_year
ORDER BY supp_nation, cust_nation, l_year;

-- Query 8: National Market Share
SELECT
    o_year,
    sum(case
        when nation = 'BRAZIL' then volume
        else 0
    end) / sum(volume) as mkt_share
FROM (
    SELECT
        extract(year from o_orderdate) as o_year,
        l_extendedprice * (1 - l_discount) as volume,
        n2.n_name as nation
    FROM part, supplier, lineitem, orders, customer, nation n1, nation n2, region
    WHERE
        p_partkey = l_partkey
        AND s_suppkey = l_suppkey
        AND l_orderkey = o_orderkey
        AND o_custkey = c_custkey
        AND c_nationkey = n1.n_nationkey
        AND n1.n_regionkey = r_regionkey
        AND r_name = 'AMERICA'
        AND s_nationkey = n2.n_nationkey
        AND o_orderdate between date '1995-01-01' AND date '1996-12-31'
        AND p_type = 'ECONOMY ANODIZED STEEL'
) as all_nations
GROUP BY o_year
ORDER BY o_year;

-- Query 9: Product Type Profit Measure
SELECT
    nation,
    o_year,
    sum(amount) as sum_profit
FROM (
    SELECT
        n_name as nation,
        extract(year from o_orderdate) as o_year,
        l_extendedprice * (1 - l_discount) - ps_supplycost * l_quantity as amount
    FROM part, supplier, lineitem, partsupp, orders, nation
    WHERE
        s_suppkey = l_suppkey
        AND ps_suppkey = l_suppkey
        AND ps_partkey = l_partkey
        AND p_partkey = l_partkey
        AND o_orderkey = l_orderkey
        AND s_nationkey = n_nationkey
        AND p_name LIKE '%green%'
) as profit
GROUP BY nation, o_year
ORDER BY nation, o_year desc;

-- Query 10: Returned Item Reporting
SELECT
    c_custkey,
    c_name,
    sum(l_extendedprice * (1 - l_discount)) as revenue,
    c_acctbal,
    n_name,
    c_address,
    c_phone,
    c_comment
FROM customer, orders, lineitem, nation
WHERE
    c_custkey = o_custkey
    AND l_orderkey = o_orderkey
    AND o_orderdate >= date '1993-10-01'
    AND o_orderdate < date '1993-10-01' + interval '3' month
    AND l_returnflag = 'R'
    AND c_nationkey = n_nationkey
GROUP BY c_custkey, c_name, c_acctbal, c_phone, n_name, c_address, c_comment
ORDER BY revenue desc
LIMIT 20;

-- Query 11: Important Stock Identification
SELECT
    ps_partkey,
    sum(ps_supplycost * ps_availqty) as value
FROM partsupp, supplier, nation
WHERE
    ps_suppkey = s_suppkey
    AND s_nationkey = n_nationkey
    AND n_name = 'GERMANY'
GROUP BY ps_partkey
HAVING sum(ps_supplycost * ps_availqty) > (
    SELECT sum(ps_supplycost * ps_availqty) * 0.0001
    FROM partsupp, supplier, nation
    WHERE
        ps_suppkey = s_suppkey
        AND s_nationkey = n_nationkey
        AND n_name = 'GERMANY'
)
ORDER BY value desc;

-- Query 12: Shipping Modes and Order Priority
SELECT
    l_shipmode,
    sum(case
        when o_orderpriority = '1-URGENT' OR o_orderpriority = '2-HIGH'
        then 1
        else 0
    end) as high_line_count,
    sum(case
        when o_orderpriority <> '1-URGENT' AND o_orderpriority <> '2-HIGH'
        then 1
        else 0
    end) as low_line_count
FROM orders, lineitem
WHERE
    o_orderkey = l_orderkey
    AND l_shipmode IN ('MAIL', 'SHIP')
    AND l_commitdate < l_receiptdate
    AND l_shipdate < l_commitdate
    AND l_receiptdate >= date '1994-01-01'
    AND l_receiptdate < date '1994-01-01' + interval '1' year
GROUP BY l_shipmode
ORDER BY l_shipmode;

-- Query 13: Customer Distribution
SELECT
    c_count,
    count(*) as custdist
FROM (
    SELECT
        c_custkey,
        count(o_orderkey) as c_count
    FROM customer left outer join orders on
        c_custkey = o_custkey
        AND o_comment not like '%special%requests%'
    GROUP BY c_custkey
) as c_orders
GROUP BY c_count
ORDER BY custdist desc, c_count desc;

-- Query 14: Promotion Effect
SELECT
    100.00 * sum(case
        when p_type like 'PROMO%'
        then l_extendedprice * (1 - l_discount)
        else 0
    end) / sum(l_extendedprice * (1 - l_discount)) as promo_revenue
FROM lineitem, part
WHERE
    l_partkey = p_partkey
    AND l_shipdate >= date '1995-09-01'
    AND l_shipdate < date '1995-09-01' + interval '1' month;

-- Query 15: Top Supplier
CREATE VIEW revenue (supplier_no, total_revenue) AS
SELECT
    l_suppkey,
    sum(l_extendedprice * (1 - l_discount))
FROM lineitem
WHERE
    l_shipdate >= date '1996-01-01'
    AND l_shipdate < date '1996-01-01' + interval '3' month
GROUP BY l_suppkey;

SELECT
    s_suppkey,
    s_name,
    s_address,
    s_phone,
    total_revenue
FROM supplier, revenue
WHERE
    s_suppkey = supplier_no
    AND total_revenue = (
        SELECT max(total_revenue)
        FROM revenue
    )
ORDER BY s_suppkey;

DROP VIEW revenue;

-- Query 16: Parts/Supplier Relationship
SELECT
    p_brand,
    p_type,
    p_size,
    count(distinct ps_suppkey) as supplier_cnt
FROM partsupp, part
WHERE
    p_partkey = ps_partkey
    AND p_brand <> 'Brand#45'
    AND p_type not like 'MEDIUM POLISHED%'
    AND p_size IN (49, 14, 23, 45, 19, 3, 36, 9)
    AND ps_suppkey not in (
        SELECT s_suppkey
        FROM supplier
        WHERE s_comment like '%Customer%Complaints%'
    )
GROUP BY p_brand, p_type, p_size
ORDER BY supplier_cnt desc, p_brand, p_type, p_size;

-- Query 17: Small-Quantity-Order Revenue
SELECT
    sum(l_extendedprice) / 7.0 as avg_yearly
FROM lineitem, part
WHERE
    p_partkey = l_partkey
    AND p_brand = 'Brand#23'
    AND p_container = 'MED BOX'
    AND l_quantity < (
        SELECT 0.2 * avg(l_quantity)
        FROM lineitem
        WHERE l_partkey = p_partkey
    );

-- Query 18: Large Volume Customer
SELECT
    c_name,
    c_custkey,
    o_orderkey,
    o_orderdate,
    o_totalprice,
    sum(l_quantity)
FROM customer, orders, lineitem
WHERE
    o_orderkey IN (
        SELECT l_orderkey
        FROM lineitem
        GROUP BY l_orderkey
        HAVING sum(l_quantity) > 300
    )
    AND c_custkey = o_custkey
    AND o_orderkey = l_orderkey
GROUP BY c_name, c_custkey, o_orderkey, o_orderdate, o_totalprice
ORDER BY o_totalprice desc, o_orderdate
LIMIT 100;

-- Query 19: Discounted Revenue
SELECT
    sum(l_extendedprice * (1 - l_discount)) as revenue
FROM lineitem, part
WHERE
    (
        p_partkey = l_partkey
        AND p_brand = 'Brand#12'
        AND p_container IN ('SM CASE', 'SM BOX', 'SM PACK', 'SM PKG')
        AND l_quantity >= 1 and l_quantity <= 1 + 10
        AND p_size between 1 and 5
        AND l_shipmode IN ('AIR', 'AIR REG')
        AND l_shipinstruct = 'DELIVER IN PERSON'
    )
    OR
    (
        p_partkey = l_partkey
        AND p_brand = 'Brand#23'
        AND p_container IN ('MED BAG', 'MED BOX', 'MED PKG', 'MED PACK')
        AND l_quantity >= 10 and l_quantity <= 10 + 10
        AND p_size between 1 and 10
        AND l_shipmode IN ('AIR', 'AIR REG')
        AND l_shipinstruct = 'DELIVER IN PERSON'
    )
    OR
    (
        p_partkey = l_partkey
        AND p_brand = 'Brand#34'
        AND p_container IN ('LG CASE', 'LG BOX', 'LG PACK', 'LG PKG')
        AND l_quantity >= 20 and l_quantity <= 20 + 10
        AND p_size between 1 and 15
        AND l_shipmode IN ('AIR', 'AIR REG')
        AND l_shipinstruct = 'DELIVER IN PERSON'
    );

-- Query 20: Potential Part Promotion
SELECT
    s_name,
    s_address
FROM supplier, nation
WHERE
    s_suppkey IN (
        SELECT ps_suppkey
        FROM partsupp
        WHERE ps_partkey IN (
            SELECT p_partkey
            FROM part
            WHERE p_name like 'forest%'
        )
        AND ps_availqty > (
            SELECT 0.5 * sum(l_quantity)
            FROM lineitem
            WHERE
                l_partkey = ps_partkey
                AND l_suppkey = ps_suppkey
                AND l_shipdate >= date '1994-01-01'
                AND l_shipdate < date '1994-01-01' + interval '1' year
        )
    )
    AND s_nationkey = n_nationkey
    AND n_name = 'CANADA'
ORDER BY s_name;

-- Query 21: Suppliers Who Kept Orders Waiting
SELECT
    s_name,
    count(*) as numwait
FROM supplier, lineitem l1, orders, nation
WHERE
    s_suppkey = l1.l_suppkey
    AND o_orderkey = l1.l_orderkey
    AND o_orderstatus = 'F'
    AND l1.l_receiptdate > l1.l_commitdate
    AND EXISTS (
        SELECT *
        FROM lineitem l2
        WHERE l2.l_orderkey = l1.l_orderkey
            AND l2.l_suppkey <> l1.l_suppkey
    )
    AND NOT EXISTS (
        SELECT *
        FROM lineitem l3
        WHERE l3.l_orderkey = l1.l_orderkey
            AND l3.l_suppkey <> l1.l_suppkey
            AND l3.l_receiptdate > l3.l_commitdate
    )
    AND s_nationkey = n_nationkey
    AND n_name = 'SAUDI ARABIA'
GROUP BY s_name
ORDER BY numwait desc, s_name
LIMIT 100;

-- Query 22: Global Sales Opportunity
SELECT
    cntrycode,
    count(*) as numcust,
    sum(c_acctbal) as totacctbal
FROM (
    SELECT
        substring(c_phone from 1 for 2) as cntrycode,
        c_acctbal
    FROM customer
    WHERE
        substring(c_phone from 1 for 2) IN ('13', '31', '23', '29', '30', '18', '17')
        AND c_acctbal > (
            SELECT avg(c_acctbal)
            FROM customer
            WHERE
                c_acctbal > 0.00
                AND substring(c_phone from 1 for 2) IN ('13', '31', '23', '29', '30', '18', '17')
        )
        AND NOT EXISTS (
            SELECT *
            FROM orders
            WHERE o_custkey = c_custkey
        )
) as custsale
GROUP BY cntrycode
ORDER BY cntrycode;
