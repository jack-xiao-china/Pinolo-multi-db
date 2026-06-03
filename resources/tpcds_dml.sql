-- TPC-DS 99 Standard Queries
-- TPC Benchmark DS v3.2.0
-- Note: These are simplified versions focusing on core query patterns

-- Query 1: Store returns analysis
SELECT
    cd_gender,
    cd_marital_status,
    cd_education_status,
    count(*) cnt1,
    cd_purchase_estimate,
    count(*) cnt2,
    cd_credit_rating,
    count(*) cnt3
FROM customer c, customer_address ca, customer_demographics
WHERE
    c.c_current_addr_sk = ca.ca_address_sk
    AND ca_state IN ('KY','GA','NM')
    AND cd_demo_sk = c.c_current_cdemo_sk
    AND EXISTS (
        SELECT * FROM store_sales, date_dim
        WHERE c.c_customer_sk = ss_customer_sk
            AND ss_sold_date_sk = d_date_sk
            AND d_year = 2001
            AND d_moy BETWEEN 4 AND 10
    )
    AND NOT EXISTS (
        SELECT * FROM web_sales, date_dim
        WHERE c.c_customer_sk = ws_bill_customer_sk
            AND ws_sold_date_sk = d_date_sk
            AND d_year = 2001
            AND d_moy BETWEEN 4 AND 10
    )
GROUP BY cd_gender, cd_marital_status, cd_education_status, cd_purchase_estimate, cd_credit_rating
ORDER BY cd_gender, cd_marital_status, cd_education_status, cd_purchase_estimate, cd_credit_rating
LIMIT 100;

-- Query 2: Inventory analysis
SELECT
    w_state,
    i_item_id,
    sum(CASE WHEN d_date < '2001-05-02' THEN cs_sales_price - coalesce(cr_refunded_cash,0) ELSE 0 END) AS sales_before,
    sum(CASE WHEN d_date >= '2001-05-02' THEN cs_sales_price - coalesce(cr_refunded_cash,0) ELSE 0 END) AS sales_after
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

-- Query 3: Store sales by promotion
SELECT
    dt.d_year,
    item.i_category_id,
    item.i_category,
    sum(ss_ext_sales_price) AS total_sales
FROM store_sales
JOIN date_dim dt ON ss_sold_date_sk = dt.d_date_sk
JOIN item ON ss_item_sk = item.i_item_sk
WHERE item.i_manager_id = 1
    AND dt.d_moy = 11
    AND dt.d_year = 2000
GROUP BY dt.d_year, item.i_category_id, item.i_category
ORDER BY total_sales DESC, dt.d_year, item.i_category_id, item.i_category
LIMIT 100;

-- Query 4: Customer shopping patterns
SELECT
    c_last_name,
    c_first_name,
    ca_city,
    bought_city,
    ss_ticket_number,
    extended_price,
    extended_tax,
    list_price
FROM (
    SELECT
        ss_ticket_number,
        ss_customer_sk,
        ca_city AS bought_city,
        sum(ss_ext_sales_price) AS extended_price,
        sum(ss_ext_list_price) AS list_price,
        sum(ss_ext_tax) AS extended_tax
    FROM store_sales, date_dim, store, household_demographics, customer_address
    WHERE ss_sold_date_sk = d_date_sk
        AND ss_store_sk = s_store_sk
        AND ss_hdemo_sk = hd_demo_sk
        AND ss_addr_sk = ca_address_sk
        AND d_dom BETWEEN 1 AND 2
        AND (hd_dep_count = 4 OR hd_vehicle_count = 3)
        AND d_year IN (1999,2000,2001)
        AND s_city IN ('Fairview','Midway')
    GROUP BY ss_ticket_number, ss_customer_sk, ca_city
) dn
JOIN customer ON ss_customer_sk = c_customer_sk
JOIN customer_address ON c_current_addr_sk = ca_address_sk
WHERE ca_city <> bought_city
ORDER BY c_last_name, ss_ticket_number
LIMIT 100;

-- Query 5: Catalog sales analysis
SELECT
    substr(w_zip,1,2) || substr(w_zip,3,3) AS zip_code,
    count(cs1.cs_order_number) AS cnt,
    sum(cs1.cs_ext_sales_price) AS sales
FROM catalog_sales cs1
JOIN date_dim ON cs1.cs_sold_date_sk = d_date_sk
JOIN customer_address ON cs1.cs_bill_addr_sk = ca_address_sk
JOIN warehouse ON cs1.cs_warehouse_sk = w_warehouse_sk
WHERE d_year = 2002
    AND d_moy = 2
    AND ca_zip IN ('85669','86197','88274','83405','86475','85392','85460','80348','81792')
GROUP BY substr(w_zip,1,2) || substr(w_zip,3,3)
ORDER BY zip_code, cnt, sales
LIMIT 100;

-- Query 6: Web sales profitability
SELECT
    i_brand_id,
    i_brand,
    i_manufact_id,
    i_manufact,
    sum(ss_ext_sales_price) AS ext_price
FROM store_sales, item, customer, customer_address, date_dim
WHERE ss_item_sk = i_item_sk
    AND ss_customer_sk = c_customer_sk
    AND c_current_addr_sk = ca_address_sk
    AND ss_sold_date_sk = d_date_sk
    AND d_date BETWEEN '2000-01-01' AND '2000-12-31'
    AND i_category = 'Books'
GROUP BY i_brand_id, i_brand, i_manufact_id, i_manufact
ORDER BY ext_price DESC, i_brand_id, i_brand, i_manufact_id, i_manufact
LIMIT 100;

-- Query 7: Store returns by reason
SELECT
    r_reason_desc,
    count(*) AS return_count,
    sum(sr_return_amt) AS total_return
FROM store_returns, reason, date_dim
WHERE sr_reason_sk = r_reason_sk
    AND sr_returned_date_sk = d_date_sk
    AND d_year = 2001
GROUP BY r_reason_desc
ORDER BY return_count DESC, total_return DESC
LIMIT 100;

-- Query 8: Catalog returns analysis
SELECT
    i_item_id,
    sum(cs_sales_price) AS sales,
    sum(cr_return_amount) AS returns,
    sum(cs_net_profit - coalesce(cr_net_loss,0)) AS profit
FROM catalog_sales
LEFT OUTER JOIN catalog_returns ON cs_item_sk = cr_item_sk AND cs_order_number = cr_order_number
JOIN item ON cs_item_sk = i_item_sk
JOIN date_dim ON cs_sold_date_sk = d_date_sk
WHERE d_year = 2001
    AND d_moy = 8
GROUP BY i_item_id
ORDER BY sales DESC
LIMIT 100;

-- Query 9: Web returns by customer demographics
SELECT
    cd_gender,
    cd_marital_status,
    cd_education_status,
    count(*) AS cnt1,
    sum(wr_return_amt) AS total_return
FROM web_returns, customer, customer_demographics, date_dim
WHERE wr_refunded_customer_sk = c_customer_sk
    AND c_current_cdemo_sk = cd_demo_sk
    AND wr_returned_date_sk = d_date_sk
    AND d_year = 2001
GROUP BY cd_gender, cd_marital_status, cd_education_status
ORDER BY cd_gender, cd_marital_status, cd_education_status
LIMIT 100;

-- Query 10: Store sales by customer demographics
SELECT
    cd_gender,
    cd_marital_status,
    cd_education_status,
    count(*) AS cnt1,
    count(*) AS cnt2,
    count(*) AS cnt3,
    sum(ss_ext_sales_price) AS total_sales
FROM store_sales, customer, customer_demographics, date_dim
WHERE ss_customer_sk = c_customer_sk
    AND c_current_cdemo_sk = cd_demo_sk
    AND ss_sold_date_sk = d_date_sk
    AND d_year = 1999
    AND d_moy = 11
GROUP BY cd_gender, cd_marital_status, cd_education_status
ORDER BY cd_gender, cd_marital_status, cd_education_status
LIMIT 100;

-- Query 11: Customer web vs store comparison
SELECT
    c_last_name,
    c_first_name,
    sum(case when (channel = 'web') then sales else 0 end) AS web_sales,
    sum(case when (channel = 'store') then sales else 0 end) AS store_sales,
    sum(case when (channel = 'catalog') then sales else 0 end) AS catalog_sales
FROM (
    SELECT c_last_name, c_first_name, 'web' AS channel, sum(ws_ext_sales_price) AS sales
    FROM web_sales, customer, date_dim
    WHERE ws_bill_customer_sk = c_customer_sk
        AND ws_sold_date_sk = d_date_sk
        AND d_year = 2000
        AND d_moy = 11
    GROUP BY c_last_name, c_first_name
    UNION ALL
    SELECT c_last_name, c_first_name, 'store' AS channel, sum(ss_ext_sales_price) AS sales
    FROM store_sales, customer, date_dim
    WHERE ss_customer_sk = c_customer_sk
        AND ss_sold_date_sk = d_date_sk
        AND d_year = 2000
        AND d_moy = 11
    GROUP BY c_last_name, c_first_name
    UNION ALL
    SELECT c_last_name, c_first_name, 'catalog' AS channel, sum(cs_ext_sales_price) AS sales
    FROM catalog_sales, customer, date_dim
    WHERE cs_bill_customer_sk = c_customer_sk
        AND cs_sold_date_sk = d_date_sk
        AND d_year = 2000
        AND d_moy = 11
    GROUP BY c_last_name, c_first_name
) x
GROUP BY c_last_name, c_first_name
ORDER BY c_last_name, c_first_name, web_sales
LIMIT 100;

-- Query 12: Store sales by time
SELECT
    t_hour,
    t_minute,
    count(*) AS cnt
FROM store_sales, time_dim, store
WHERE ss_sold_time_sk = t_time_sk
    AND ss_store_sk = s_store_sk
    AND s_store_name = 'Store#1'
GROUP BY t_hour, t_minute
ORDER BY t_hour, t_minute
LIMIT 100;

-- Query 13: Catalog sales by category
SELECT
    i_category,
    i_class,
    i_brand,
    i_product_name,
    d_year,
    d_qoy,
    d_moy,
    s_store_id,
    sum(ss_sales_price * ss_quantity) AS sales
FROM store_sales, item, date_dim, store
WHERE ss_item_sk = i_item_sk
    AND ss_sold_date_sk = d_date_sk
    AND ss_store_sk = s_store_sk
    AND d_year = 2000
GROUP BY i_category, i_class, i_brand, i_product_name, d_year, d_qoy, d_moy, s_store_id
ORDER BY sales DESC
LIMIT 100;

-- Query 14: Store returns by store
SELECT
    s_store_name,
    s_company_id,
    count(*) AS num_returns,
    sum(sr_return_amt) AS total_returns
FROM store_returns, store, date_dim
WHERE sr_store_sk = s_store_sk
    AND sr_returned_date_sk = d_date_sk
    AND d_year = 2001
    AND d_moy = 11
GROUP BY s_store_name, s_company_id
ORDER BY total_returns DESC
LIMIT 100;

-- Query 15: Web sales by web page
SELECT
    wp_web_page_id,
    count(*) AS cnt,
    sum(ws_ext_sales_price) AS sales
FROM web_sales, web_page, date_dim
WHERE ws_web_page_sk = wp_web_page_sk
    AND ws_sold_date_sk = d_date_sk
    AND d_year = 2000
    AND d_moy = 11
GROUP BY wp_web_page_id
ORDER BY sales DESC
LIMIT 100;

-- Query 16: Catalog sales by catalog page
SELECT
    cp_catalog_page_id,
    count(*) AS cnt,
    sum(cs_ext_sales_price) AS sales
FROM catalog_sales, catalog_page, date_dim
WHERE cs_catalog_page_sk = cp_catalog_page_sk
    AND cs_sold_date_sk = d_date_sk
    AND d_year = 2000
    AND d_moy = 11
GROUP BY cp_catalog_page_id
ORDER BY sales DESC
LIMIT 100;

-- Query 17: Store sales by promotion channel
SELECT
    p_channel_dmail,
    p_channel_email,
    p_channel_tv,
    count(*) AS cnt,
    sum(ss_ext_sales_price) AS sales
FROM store_sales, promotion, date_dim
WHERE ss_promo_sk = p_promo_sk
    AND ss_sold_date_sk = d_date_sk
    AND d_year = 2000
    AND d_moy = 11
GROUP BY p_channel_dmail, p_channel_email, p_channel_tv
ORDER BY sales DESC
LIMIT 100;

-- Query 18: Customer address analysis
SELECT
    ca_state,
    count(*) AS cnt,
    avg(c_acctbal) AS avg_acctbal
FROM customer, customer_address
WHERE c_current_addr_sk = ca_address_sk
    AND ca_state IN ('CA','NY','TX')
GROUP BY ca_state
ORDER BY cnt DESC
LIMIT 100;

-- Query 19: Item profitability
SELECT
    i_item_id,
    i_item_desc,
    i_category,
    i_class,
    i_current_price,
    sum(ss_ext_sales_price) AS itemrevenue,
    sum(ss_ext_sales_price)*100/sum(sum(ss_ext_sales_price)) OVER () AS revenueratio
FROM store_sales, item, date_dim
WHERE ss_item_sk = i_item_sk
    AND i_category = 'Jewelry'
    AND ss_sold_date_sk = d_date_sk
    AND d_date BETWEEN '2000-05-01' AND '2000-05-31'
GROUP BY i_item_id, i_item_desc, i_category, i_class, i_current_price
ORDER BY itemrevenue DESC
LIMIT 100;

-- Query 20: Store sales by geography
SELECT
    ca_zip,
    count(*) AS cnt,
    sum(ss_ext_sales_price) AS sales
FROM store_sales, customer_address, date_dim
WHERE ss_addr_sk = ca_address_sk
    AND ss_sold_date_sk = d_date_sk
    AND d_year = 2001
    AND d_qoy = 1
GROUP BY ca_zip
ORDER BY cnt DESC
LIMIT 100;

-- Query 21-99: Additional complex queries omitted for brevity
-- (Full TPC-DS includes 99 queries covering various analytical patterns)
