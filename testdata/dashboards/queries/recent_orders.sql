SELECT id, customer_name, amount, status, created_at
FROM orders
WHERE created_at >= '{{ filters.date_range.start }}'
  AND created_at <= '{{ filters.date_range.end }}'
ORDER BY created_at DESC
LIMIT 25
