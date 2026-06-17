# Результаты нагрузочных тестов ручки поиска анкеты

## Запрос

```sql
SELECT id, first_name, second_name, birthdate, COALESCE(biography, ''), city
FROM users
WHERE first_name ILIKE $1
	AND second_name ILIKE $2
ORDER BY id ASC
```

## Сравнение показателей до и после создания индекса

| N clients | Latency before (avg) | Latency after | Throughput before (req/s)| Throughput after|
|-----|----------------|---------------|-------------------|------------------|
|1|82ms|0.9ms|12.19|1136|
|10|168ms|2.2ms|59.1|4601|
|100|1609ms|14ms|59.99|6957|
|1000|8367ms|111ms|74.38|8325|

<details>
  <summary><b>Детали результатов</b></summary>

### BEFORE
<details>
<summary>1</summary>

	Summary:
	Total:        20.0144 secs
	Slowest:      0.1041 secs
	Fastest:      0.0761 secs
	Average:      0.0820 secs
	Requests/sec: 12.1912
	

	Response time histogram:
	0.076 [1]     |
	0.079 [24]    |■■■■■■■■■■
	0.082 [87]    |■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■
	0.084 [92]    |■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■
	0.087 [35]    |■■■■■■■■■■■■■■■
	0.090 [3]     |■
	0.093 [1]     |
	0.096 [0]     |
	0.099 [0]     |
	0.101 [0]     |
	0.104 [1]     |


	Latency distribution:
	10%% in 0.0789 secs
	25%% in 0.0800 secs
	50%% in 0.0820 secs
	75%% in 0.0837 secs
	90%% in 0.0854 secs
	95%% in 0.0866 secs
	99%% in 0.0903 secs

	Details (average, fastest, slowest):
	DNS+dialup:   0.0000 secs, 0.0000 secs, 0.0014 secs
	DNS-lookup:   0.0000 secs, 0.0000 secs, 0.0011 secs
	req write:    0.0000 secs, 0.0000 secs, 0.0000 secs
	resp wait:    0.0819 secs, 0.0760 secs, 0.1026 secs
	resp read:    0.0001 secs, 0.0000 secs, 0.0002 secs

	Status code distribution:
	[200] 244 responses
</details>
<details>
<summary>10</summary>

	Summary:
	Total:        20.2011 secs
	Slowest:      0.3176 secs
	Fastest:      0.0885 secs
	Average:      0.1681 secs
	Requests/sec: 59.1058
	

	Response time histogram:
	0.089 [1]     |
	0.111 [477]   |■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■
	0.134 [22]    |■■
	0.157 [292]   |■■■■■■■■■■■■■■■■■■■■■■■■
	0.180 [5]     |
	0.203 [0]     |
	0.226 [0]     |
	0.249 [1]     |
	0.272 [181]   |■■■■■■■■■■■■■■■
	0.295 [206]   |■■■■■■■■■■■■■■■■■
	0.318 [9]     |■


	Latency distribution:
	10%% in 0.0959 secs
	25%% in 0.0993 secs
	50%% in 0.1391 secs
	75%% in 0.2679 secs
	90%% in 0.2784 secs
	95%% in 0.2835 secs
	99%% in 0.2937 secs

	Details (average, fastest, slowest):
	DNS+dialup:   0.0000 secs, 0.0000 secs, 0.0020 secs
	DNS-lookup:   0.0000 secs, 0.0000 secs, 0.0011 secs
	req write:    0.0000 secs, 0.0000 secs, 0.0001 secs
	resp wait:    0.1680 secs, 0.0885 secs, 0.3175 secs
	resp read:    0.0001 secs, 0.0000 secs, 0.0031 secs

	Status code distribution:
	[200] 1194 responses
</details>
<details>
	<summary>100</summary>

	Summary:
	Total:        21.5839 secs
	Slowest:      11.8838 secs
	Fastest:      0.1745 secs
	Average:      1.6092 secs
	Requests/sec: 59.9985
	

	Response time histogram:
	0.174 [1]     |
	1.345 [673]   |■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■
	2.516 [404]   |■■■■■■■■■■■■■■■■■■■■■■■■
	3.687 [141]   |■■■■■■■■
	4.858 [45]    |■■■
	6.029 [18]    |■
	7.200 [6]     |
	8.371 [4]     |
	9.542 [1]     |
	10.713 [1]    |
	11.884 [1]    |


	Latency distribution:
	10%% in 0.5279 secs
	25%% in 0.7612 secs
	50%% in 1.3105 secs
	75%% in 2.0583 secs
	90%% in 3.1184 secs
	95%% in 3.8423 secs
	99%% in 6.3721 secs

	Details (average, fastest, slowest):
	DNS+dialup:   0.0006 secs, 0.0000 secs, 0.0092 secs
	DNS-lookup:   0.0001 secs, 0.0000 secs, 0.0019 secs
	req write:    0.0000 secs, 0.0000 secs, 0.0000 secs
	resp wait:    1.6075 secs, 0.1744 secs, 11.8801 secs
	resp read:    0.0010 secs, 0.0000 secs, 0.0113 secs

	Status code distribution:
	[200] 1295 responses
</details>
<details>
<summary>1000</summary>

	Summary:
	Total:        30.0719 secs
	Slowest:      19.9616 secs
	Fastest:      0.1927 secs
	Average:      8.3671 secs
	Requests/sec: 74.3884
	

	Response time histogram:
	0.193 [1]     |
	2.170 [267]   |■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■
	4.146 [234]   |■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■
	6.123 [237]   |■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■
	8.100 [217]   |■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■
	10.077 [199]  |■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■
	12.054 [179]  |■■■■■■■■■■■■■■■■■■■■■■■■■■■
	14.031 [166]  |■■■■■■■■■■■■■■■■■■■■■■■■■
	16.008 [124]  |■■■■■■■■■■■■■■■■■■■
	17.985 [106]  |■■■■■■■■■■■■■■■■
	19.962 [98]   |■■■■■■■■■■■■■■■


	Latency distribution:
	10%% in 1.5842 secs
	25%% in 3.7923 secs
	50%% in 7.7011 secs
	75%% in 12.4887 secs
	90%% in 16.3696 secs
	95%% in 18.0587 secs
	99%% in 19.4798 secs

	Details (average, fastest, slowest):
	DNS+dialup:   0.0257 secs, 0.0000 secs, 0.1113 secs
	DNS-lookup:   0.0026 secs, 0.0000 secs, 0.0166 secs
	req write:    0.0000 secs, 0.0000 secs, 0.0016 secs
	resp wait:    8.3404 secs, 0.1926 secs, 19.9612 secs
	resp read:    0.0009 secs, 0.0000 secs, 0.0139 secs

	Status code distribution:
	[200] 1828 responses

	Error distribution:
	[409] Get "http://localhost:8080/api/v1/user/search?first_name=Александр&last_name=Абрамов": context deadline exceeded (Client.Timeout exceeded while awaiting headers)

</details>

### AFTER
<details>
<summary>1</summary>
	Summary:
	Total:        20.0008 secs
	Slowest:      0.0163 secs
	Fastest:      0.0007 secs
	Average:      0.0009 secs
	Requests/sec: 1136.8561
	

	Response time histogram:
	0.001 [1]     |
	0.002 [22577] |■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■
	0.004 [151]   |
	0.005 [4]     |
	0.007 [2]     |
	0.009 [2]     |
	0.010 [0]     |
	0.012 [0]     |
	0.013 [0]     |
	0.015 [0]     |
	0.016 [1]     |


	Latency distribution:
	10%% in 0.0008 secs
	25%% in 0.0008 secs
	50%% in 0.0008 secs
	75%% in 0.0008 secs
	90%% in 0.0010 secs
	95%% in 0.0015 secs
	99%% in 0.0021 secs

	Details (average, fastest, slowest):
	DNS+dialup:   0.0000 secs, 0.0000 secs, 0.0031 secs
	DNS-lookup:   0.0000 secs, 0.0000 secs, 0.0014 secs
	req write:    0.0000 secs, 0.0000 secs, 0.0002 secs
	resp wait:    0.0008 secs, 0.0006 secs, 0.0124 secs
	resp read:    0.0000 secs, 0.0000 secs, 0.0037 secs

	Status code distribution:
	[200] 22738 responses
</details>

<details>
<summary>10</summary>

	Summary:
	Total:        20.0013 secs
	Slowest:      0.0189 secs
	Fastest:      0.0008 secs
	Average:      0.0022 secs
	Requests/sec: 4601.8574
	

	Response time histogram:
	0.001 [1]     |
	0.003 [67897] |■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■
	0.004 [23186] |■■■■■■■■■■■■■■
	0.006 [903]   |■
	0.008 [44]    |
	0.010 [1]     |
	0.012 [1]     |
	0.013 [0]     |
	0.015 [3]     |
	0.017 [4]     |
	0.019 [3]     |


	Latency distribution:
	10%% in 0.0014 secs
	25%% in 0.0016 secs
	50%% in 0.0020 secs
	75%% in 0.0026 secs
	90%% in 0.0032 secs
	95%% in 0.0035 secs
	99%% in 0.0044 secs

	Details (average, fastest, slowest):
	DNS+dialup:   0.0000 secs, 0.0000 secs, 0.0021 secs
	DNS-lookup:   0.0000 secs, 0.0000 secs, 0.0012 secs
	req write:    0.0000 secs, 0.0000 secs, 0.0002 secs
	resp wait:    0.0020 secs, 0.0008 secs, 0.0163 secs
	resp read:    0.0001 secs, 0.0000 secs, 0.0044 secs

	Status code distribution:
	[200] 92043 responses
</details>

<details>
<summary>100</summary>

	Summary:
	Total:        20.0131 secs
	Slowest:      0.0937 secs
	Fastest:      0.0012 secs
	Average:      0.0144 secs
	Requests/sec: 6957.3992
	

	Response time histogram:
	0.001 [1]     |
	0.010 [51755] |■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■
	0.020 [60200] |■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■
	0.029 [19176] |■■■■■■■■■■■■■
	0.038 [5702]  |■■■■
	0.047 [1689]  |■
	0.057 [476]   |
	0.066 [168]   |
	0.075 [53]    |
	0.084 [14]    |
	0.094 [5]     |


	Latency distribution:
	10%% in 0.0063 secs
	25%% in 0.0087 secs
	50%% in 0.0125 secs
	75%% in 0.0179 secs
	90%% in 0.0249 secs
	95%% in 0.0302 secs
	99%% in 0.0424 secs

	Details (average, fastest, slowest):
	DNS+dialup:   0.0000 secs, 0.0000 secs, 0.0074 secs
	DNS-lookup:   0.0000 secs, 0.0000 secs, 0.0019 secs
	req write:    0.0000 secs, 0.0000 secs, 0.0005 secs
	resp wait:    0.0141 secs, 0.0012 secs, 0.0936 secs
	resp read:    0.0003 secs, 0.0000 secs, 0.0163 secs

	Status code distribution:
	[200] 139239 responses
</details>

<details>
<summary>1000</summary>

	Summary:
	Total:        21.7088 secs
	Slowest:      5.6606 secs
	Fastest:      0.0094 secs
	Average:      0.1110 secs
	Requests/sec: 8325.1585
	

	Response time histogram:
	0.009 [1]     |
	0.574 [178341]        |■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■
	1.140 [2084]  |
	1.705 [212]   |
	2.270 [80]    |
	2.835 [6]     |
	3.400 [1]     |
	3.965 [3]     |
	4.530 [0]     |
	5.095 [0]     |
	5.661 [1]     |


	Latency distribution:
	10%% in 0.0408 secs
	25%% in 0.0476 secs
	50%% in 0.0641 secs
	75%% in 0.1135 secs
	90%% in 0.2206 secs
	95%% in 0.3754 secs
	99%% in 0.6691 secs

	Details (average, fastest, slowest):
	DNS+dialup:   0.0001 secs, 0.0000 secs, 0.1032 secs
	DNS-lookup:   0.0000 secs, 0.0000 secs, 0.0153 secs
	req write:    0.0000 secs, 0.0000 secs, 0.0073 secs
	resp wait:    0.0492 secs, 0.0080 secs, 3.4368 secs
	resp read:    0.0615 secs, 0.0000 secs, 4.0284 secs

	Status code distribution:
	[200] 180729 responses
</details>

</details>
<!-- |1|80ms|0.9ms|12.31 req/s|1171 req/s|0|0|
|10|160ms|4.1ms|58.5 req/s|2462 req/s|0|0|
|100|250ms|49ms|381.46 req/s|2037 req/s|83|32|
|1000|450ms|317ms|1848.11 req/s|3087 req/s|98|39| -->

## Индекс

```sql
CREATE INDEX idx_users_fname_sname_gin ON public.users USING gin (first_name gin_trgm_ops, second_name gin_trgm_ops)
```

## Explain запроса

```sql
Sort  (cost=300.45..300.45 rows=2 width=99) (actual time=1.259..1.265 rows=98.00 loops=1)
  Sort Key: id
  Sort Method: quicksort  Memory: 34kB
  Buffers: shared hit=105
  ->  Bitmap Heap Scan on users  (cost=292.46..300.44 rows=2 width=99) (actual time=1.017..1.214 rows=98.00 loops=1)
        Recheck Cond: ((first_name ~~* 'Александр'::text) AND (second_name ~~* 'Абрамов'::text))
        Heap Blocks: exact=33
        Buffers: shared hit=102
        ->  Bitmap Index Scan on idx_users_fname_sname_gin  (cost=0.00..292.46 rows=2 width=0) (actual time=0.998..0.998 rows=98.00 loops=1)
              Index Cond: ((first_name ~~* 'Александр'::text) AND (second_name ~~* 'Абрамов'::text))
              Index Searches: 1
              Buffers: shared hit=69
Planning:
  Buffers: shared hit=144
Planning Time: 1.331 ms
Execution Time: 1.376 ms
```
