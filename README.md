# Text Downloader

In light of the COVID-19 crisis, Springer has released several textbooks for free.
See [this news article](https://www.springernature.com/gp/librarians/news-events/all-news-articles/industry-news-initiatives/free-access-to-textbooks-for-institutions-affected-by-coronaviru/17855960) for a description of their generous offering.

So many of these textbooks looked cool, and I didn't want to download each one individually. In classic https://xkcd.com/974/ style, I wrote a script to download all of the textbooks automatically. Writing it in Golang let everything happen in parallel with minimal memory footprint. A thread for reading the CSV input (extracted from Springer's Excel spreadsheet) passes Textbook metadata to 100 workers. Each worker reads from the Springer link to find the PDF URL. Then it streams the PDF content from the web into a new file on disk.

## Usage

1. Install and run [Docker](https://www.docker.com/products/docker-desktop).
2. Clone this repo.
3. `./run` from the root of this repo.
    - Configuration: `./run -help` to see flags. For example, you can choose to download EPUBs instead of the default PDF.

## Caveats

This code is meant to be single-use.
It does rudimentary error handling by infinite retries.
It isn't robust to changes in the input CSV format, or changes to Springer landing pages.
