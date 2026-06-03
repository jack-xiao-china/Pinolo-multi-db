-- TPC-DS Core Queries for Pinolo Integration Testing
-- Selected 12 queries optimized for mutation testing (rich WHERE/HAVING/JOIN, minimal window functions)

-- Query 2: Inventory analysis (多表 JOIN, 聚合, CASE, 日期过滤)
SELECT
    w_state,
    i_item_id,
    SUM(CASE WHEN d_date < '2001-05-02' THEN cs_sales_price - COALESCE(cr_refunded_cash,0) ELSE 0 END) AS sales_before,
    SUM(CASE WHEN d_date >= '2001-05-02' THEN cs_sales_price - COALESCE(cr_refunded_cash,0) ELSE 0 END) AS sales_after
FROM catalog_sales
LEFT OUTER JOIN catalog_returns ON cs_order_number = cr_order_number AND cs_item_sk = cr_item_sk
JOIN warehouse ON cs_warehouse_sk = w_warehouse_sk
JOIN item ON i_item_sk = cs_item_sk
JOIN date_dim ON cs_sold_date_sk = d_date_sk
WHERE i_current_price BETWEEN 0.99 AND 1.49
    AND d_date BETWEEN '2001-04-02' AND '2001-06-02'
GROUP BY w_state, i_item_id
ORDER BY w_state, i_item_id
LIMIT 100;

-- Query 3: Store sales by promotion (JOIN, 聚合, 日期过滤)
SELECT
    dt.d_year,
    item.i_category_id,
    item.i_category,
    SUM(ss_ext_sales_price) AS total_sales
FROM store_sales
JOIN date_dim dt ON ss_sold_date_sk = dt.d_date_sk
JOIN item ON ss_item_sk = item.i_item_sk
WHERE item.i_manager_id = 1
    AND dt.d_moy = 11
    AND dt.d_year = 2000
GROUP BY dt.d_year, item.i_category_id, item.i_category
ORDER BY total_sales DESC, dt.d_year, item.i_category_id, item.i_category
LIMIT 100;

-- Query 5: Catalog sales analysis (JOIN, 聚合, IN 子查询)
SELECT
    SUBSTR(w_zip,1,2) || SUBSTR(w_zip,3,3) AS zip_code,
    COUNT(cs1.cs_order_number) AS cnt,
    SUM(cs1.cs_ext_sales_price) AS sales
FROM catalog_sales cs1
JOIN date_dim ON cs1.cs_sold_date_sk = d_date_sk
JOIN customer_address ON cs1.cs_bill_addr_sk = ca_address_sk
JOIN warehouse ON cs1.cs_warehouse_sk = w_warehouse_sk
WHERE d_year = 2002
    AND d_moy = 2
    AND ca_zip IN ('85669','86197','88274','83405','86475','85392','85460','80348','81792')
GROUP BY SUBSTR(w_zip,1,2) || SUBSTR(w_zip,3,3)
ORDER BY zip_code, cnt, sales
LIMIT 100;

-- Query 7: Store returns by reason (JOIN, 聚合)
SELECT
    r_reason_desc,
    COUNT(*) AS return_count,
    SUM(sr_return_amt) AS total_return
FROM store_returns
JOIN reason ON sr_reason_sk = r_reason_sk
JOIN date_dim ON sr_returned_date_sk = d_date_sk
WHERE d_year = 2001
GROUP BY r_reason_desc
ORDER BY return_count DESC, total_return DESC
LIMIT 100;

-- Query 8: Catalog returns analysis (LEFT JOIN, 聚合, 日期过滤)
SELECT
    i_item_id,
    SUM(cs_sales_price) AS sales,
    SUM(cr_return_amount) AS returns,
    SUM(cs_net_profit - COALESCE(cr_net_loss,0)) AS profit
FROM catalog_sales
LEFT OUTER JOIN catalog_returns ON cs_item_sk = cr_item_sk AND cs_order_number = cr_order_number
JOIN item ON cs_item_sk = i_item_sk
JOIN date_dim ON cs_sold_date_sk = d_date_sk
WHERE d_year = 2001
    AND d_moy = 8
GROUP BY i_item_id
ORDER BY sales DESC
LIMIT 100;

-- Query 12: Store sales by time (JOIN, 聚合, 时间过滤)
SELECT
    t_hour,
    t_minute,
    COUNT(*) AS cnt
FROM store_sales
JOIN time_dim ON ss_sold_time_sk = t_time_sk
JOIN store ON ss_store_sk = s_store_sk
WHERE s_store_name = 'Store#1'
GROUP BY t_hour, t_minute
ORDER BY t_hour, t_minute
LIMIT 100;

-- Query 14: Store returns by store (JOIN, 聚合, 日期过滤)
SELECT
    s_store_name,
    s_company_id,
    COUNT(*) AS num_returns,
    SUM(sr_return_amt) AS total_returns
FROM store_returns
JOIN store ON sr_store_sk = s_store_sk
JOIN date_dim ON sr_returned_date_sk = d_date_sk
WHERE d_year = 2001
    AND d_moy = 11
GROUP BY s_store_name, s_company_id
ORDER BY total_returns DESC
LIMIT 100;

-- Query 20: Store sales by geography (JOIN, 聚合, IN 过滤)
SELECT
    ca_zip,
    COUNT(*) AS cnt,
    SUM(ss_ext_sales_price) AS sales
FROM store_sales
JOIN customer_address ON ss_addr_sk = ca_address_sk
JOIN date_dim ON ss_sold_date_sk = d_date_sk
WHERE d_year = 2001
    AND d_qoy = 1
GROUP BY ca_zip
ORDER BY cnt DESC
LIMIT 100;

-- Query 26: Promotional sales analysis (JOIN, 聚合, 日期范围)
SELECT
    i_item_id,
    AVG(ss_quantity) AS avg_qty,
    AVG(ss_list_price) AS avg_price,
    AVG(ss_coupon_amt) AS avg_coupon,
    AVG(ss_sales_price) AS avg_sales
FROM store_sales
JOIN date_dim ON ss_sold_date_sk = d_date_sk
JOIN item ON ss_item_sk = i_item_sk
JOIN customer_demographics ON ss_cdemo_sk = cd_demo_sk
WHERE cd_gender = 'M'
    AND cd_marital_status = 'S'
    AND cd_education_status = 'College'
    AND d_year = 2000
GROUP BY i_item_id
ORDER BY i_item_id
LIMIT 100;

-- Query 32: Catalog sales promotional analysis (JOIN, 聚合, 子查询)
SELECT
    i_item_id,
    SUM(cs_ext_discount_amt) AS discount_amt
FROM catalog_sales
JOIN item ON cs_item_sk = i_item_sk
JOIN date_dim ON cs_sold_date_sk = d_date_sk
WHERE cs_item_sk IN (
    SELECT i_item_sk
    FROM item
    WHERE i_category = 'Books'
)
AND d_date BETWEEN '2000-01-01' AND '2000-03-31'
GROUP BY i_item_id
ORDER BY discount_amt DESC
LIMIT 100;

-- Query 48: Store sales quantity analysis (JOIN, 聚合, 多条件过滤)
SELECT
    SUM(ss_quantity) AS total_qty
FROM store_sales
JOIN store ON ss_store_sk = s_store_sk
JOIN customer_demographics ON ss_cdemo_sk = cd_demo_sk
JOIN customer_address ON ss_addr_sk = ca_address_sk
JOIN date_dim ON ss_sold_date_sk = d_date_sk
WHERE d_year = 2000
    AND (
        (cd_marital_status = 'M' AND cd_education_status = 'Unknown')
        OR (cd_marital_status = 'W' AND cd_education_status = 'Advanced Degree')
    )
    AND ss_sales_price BETWEEN 100.00 AND 150.00
    AND ca_country = 'United States'
    AND ca_state IN ('IN', 'OH', 'NJ');

-- Query 72: Inventory turnover analysis (JOIN, 聚合, 日期计算)
SELECT
    i_item_desc,
    w_warehouse_name,
    d1.d_week_seq,
    COUNT(CASE WHEN d2.d_date IS NOT NULL THEN 1 END) AS inv_before,
    COUNT(CASE WHEN d3.d_date IS NOT NULL THEN 1 END) AS inv_after
FROM inventory
JOIN warehouse ON inv_warehouse_sk = w_warehouse_sk
JOIN item ON inv_item_sk = i_item_sk
JOIN date_dim d1 ON inv_date_sk = d1.d_date_sk
LEFT JOIN date_dim d2 ON d2.d_date = d1.d_date - INTERVAL 30 DAY
LEFT JOIN date_dim d3 ON d3.d_date = d1.d_date + INTERVAL 30 DAY
WHERE d1.d_year = 2000
GROUP BY i_item_desc, w_warehouse_name, d1.d_week_seq
ORDER BY inv_before DESC, inv_after DESC
LIMIT 100;
