# Task

Задача:

* Реализовать web-crawler, рекурсивно скачивающий сайт (идущий по ссылкам вглубь). 
* Crawler должен скачать документ по указанному URL и продолжить закачку по ссылкам, находящимся в документе.
* Crawler должен поддерживать дозакачку.
* Crawler должен сжимать скачанные документы перед сохранением  (gzip)
* Crawler должен грузить только текстовые документы -   html, css, js (картинки, видео игнорировать)
* Crawler должен грузить документы только одного домена (игнорировать сторонние ссылки)

## Флаги запуска

```
  -d string
       	db connections string (default "postgres://postgres:postgres@postgres/crawler?sslmode=disable")
  -g bool
    	enable gzip (default true)
  -h string
       	base endpoint (default "https://github.com/chapsuk")
  -o string
       	output path (default "./result/")
  -r bool
    	resume upload
  -s bool
    	include subdomains
  -w int
       	workers count (default 150)
```

## Запуск

### Docker

Из текущей дерриктории выполняем `docker-compose up -d`, в первый раз будет собран контейнер с приложением,
после сборки запустится контейнер postgres и контейнер с приложением. Postgres нужен для поддержки дозакачки.
Результат выполнения можно посмотреть в `./result/`. Флагами запуска можно поиграться [тут](docker-compose.yml#L10).  

### Local

```
go run cmd/crawler/main.go -h http://golang-book.ru -d ""
```
для использования дозагрузки, надо приконектится к какому то постгресу, строку коннекта передавать через флаг `-d`.

## crawler vs wget (without storage)

```bash
$ time wget -r -nv -A html,css,js http://golang-book.ru

1.00 real
0.00 user
0.01 sys

$ tree golang-book.ru/
golang-book.ru/
├── assets
│   └── main.css
├── chapter-01-gettting-started.html
├── chapter-02-your-first-program.html
├── chapter-03-types.html
├── chapter-04-variables.html
├── chapter-05-control-structures.html
├── chapter-06-arrays-slices-maps.html
├── chapter-07-functions.html
├── chapter-08-pointers.html
├── chapter-09-structs-and-interfaces.html
├── chapter-10-concurrency.html
├── chapter-11-packages.html
├── chapter-12-testing.html
├── chapter-13-core-packages.html
├── chapter-14-next-steps.html
└── index.html

1 directory, 16 files
```

```bash
$ time ./crawler -h http://golang-book.ru

0.31 real
0.10 user
0.03 sys

$ tree result/golang-book.ru/
result/golang-book.ru/
├── assets
│   └── main.css.gz
├── chapter-01-gettting-started.html.gz
├── chapter-02-your-first-program.html.gz
├── chapter-03-types.html.gz
├── chapter-04-variables.html.gz
├── chapter-05-control-structures.html.gz
├── chapter-06-arrays-slices-maps.html.gz
├── chapter-07-functions.html.gz
├── chapter-08-pointers.html.gz
├── chapter-09-structs-and-interfaces.html.gz
├── chapter-10-concurrency.html.gz
├── chapter-11-packages.html.gz
├── chapter-12-testing.html.gz
├── chapter-13-core-packages.html.gz
├── chapter-14-next-steps.html.gz
└── index.html.gz

1 directory, 16 files
```

without gzip

```bash
$ time ./crawler -h http://golang-book.ru -g=0

0.27 real
0.03 user
0.01 sys

$ tree result/golang-book.ru/
result/golang-book.ru/
├── assets
│   └── main.css
├── chapter-01-gettting-started.html
├── chapter-02-your-first-program.html
├── chapter-03-types.html
├── chapter-04-variables.html
├── chapter-05-control-structures.html
├── chapter-06-arrays-slices-maps.html
├── chapter-07-functions.html
├── chapter-08-pointers.html
├── chapter-09-structs-and-interfaces.html
├── chapter-10-concurrency.html
├── chapter-11-packages.html
├── chapter-12-testing.html
├── chapter-13-core-packages.html
├── chapter-14-next-steps.html
└── index.html

1 directory, 16 files
```

with storage
```
$ time ./crawler -h http://golang-book.ru
2016/09/04 18:24:24 Completed! Time: 384.972384ms
0.39 real
0.11 user
0.03 sys
...
crawler=# select * from "golang-book.ru";
                             url                              | type | status |          created           |          updated
--------------------------------------------------------------+------+--------+----------------------------+----------------------------
 http://golang-book.ru                                        |    1 |      3 | 2016-09-04 15:24:24.155024 | 2016-09-04 15:24:24.265518
 http://golang-book.ru/chapter-01-gettting-started.html       |    1 |      3 | 2016-09-04 15:24:24.25395  | 2016-09-04 15:24:24.321782
 http://golang-book.ru/chapter-02-your-first-program.html     |    1 |      3 | 2016-09-04 15:24:24.277237 | 2016-09-04 15:24:24.371623
 http://golang-book.ru/chapter-03-types.html                  |    1 |      3 | 2016-09-04 15:24:24.280468 | 2016-09-04 15:24:24.388519
 http://golang-book.ru/chapter-04-variables.html              |    1 |      3 | 2016-09-04 15:24:24.281972 | 2016-09-04 15:24:24.39923
 http://golang-book.ru/chapter-10-concurrency.html            |    1 |      3 | 2016-09-04 15:24:24.291129 | 2016-09-04 15:24:24.423001
 http://golang-book.ru/chapter-05-control-structures.html     |    1 |      3 | 2016-09-04 15:24:24.283473 | 2016-09-04 15:24:24.433296
 http://golang-book.ru/chapter-09-structs-and-interfaces.html |    1 |      3 | 2016-09-04 15:24:24.289616 | 2016-09-04 15:24:24.441851
 http://golang-book.ru/assets/main.css                        |    2 |      3 | 2016-09-04 15:24:24.299716 | 2016-09-04 15:24:24.450094
 http://golang-book.ru/chapter-07-functions.html              |    1 |      3 | 2016-09-04 15:24:24.286585 | 2016-09-04 15:24:24.458055
 http://golang-book.ru/chapter-11-packages.html               |    1 |      3 | 2016-09-04 15:24:24.292562 | 2016-09-04 15:24:24.466465
 http://golang-book.ru/chapter-08-pointers.html               |    1 |      3 | 2016-09-04 15:24:24.288143 | 2016-09-04 15:24:24.474569
 http://golang-book.ru/chapter-06-arrays-slices-maps.html     |    1 |      3 | 2016-09-04 15:24:24.285088 | 2016-09-04 15:24:24.481355
 http://golang-book.ru/chapter-14-next-steps.html             |    1 |      3 | 2016-09-04 15:24:24.298188 | 2016-09-04 15:24:24.488203
 http://golang-book.ru/chapter-12-testing.html                |    1 |      3 | 2016-09-04 15:24:24.294398 | 2016-09-04 15:24:24.494819
 http://golang-book.ru/chapter-13-core-packages.html          |    1 |      3 | 2016-09-04 15:24:24.296463 | 2016-09-04 15:24:24.503753
 http://golang-book.ru/                                       |    1 |      3 | 2016-09-04 15:24:24.316739 | 2016-09-04 15:24:24.511351
(17 rows)

```

![](https://media.giphy.com/media/xTiTnKH3dDw1ww53R6/giphy.gif)
