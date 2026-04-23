SELECT
    DATE_TRUNC('month', created_at) AS month,
    SUM(amount) AS revenue
FROM sales
WHERE created_at >= '{{ filters.date_range.start }}'
  AND created_at <= '{{ filters.date_range.end }}'
{% if filters.region != 'All' %}
  AND region = '{{ filters.region }}'
{% endif %}
GROUP BY 1
ORDER BY 1
