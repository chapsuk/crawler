# Task


Задача:

* Реализовать web-crawler, рекурсивно скачивающий сайт (идущий по ссылкам вглубь). 
* Crawler должен скачать документ по указанному URL и продолжить закачку по ссылкам, находящимся в документе.
* Crawler должен поддерживать дозакачку.
* Crawler должен сжимать скачанные документы перед сохранением  (gzip)
* Crawler должен грузить только текстовые документы -   html, css, js (картинки, видео игнорировать)
* Crawler должен грузить документы только одного домена (игнорировать сторонние ссылки)

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
