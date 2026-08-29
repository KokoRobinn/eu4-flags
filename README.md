# EU4 Flags

A quiz on all flags in Europa Universalis 4 in the form of a web app.
If you want to run it, clone the repo and run:

`docker compose up --build -d`

It is then accessible on port 8787 or whatever port you set in `docker-compose.yml`.

## DB Overview

| id  | tag        | name        | flag_path   | capital_subcontinent | capital_region | capital_province | notes |
|:---:|:----------:|:-----------:|------------:|:--------------------:|:--------------:|:----------------:|:-----:|
| int | varchar(3) | varchar(63) | varchar(63) | varchar(63)          | varchar(63)    | varchar(63)      | TEXT  |

The `flag_path` field contains the relative file path to the image.

## Improvements

This project is far from perfect and there are many things that can be improved. 
Most pressing are the following.

### Finish UI

There is no explaination as to what you are looking at when opening the page.
Could be considered confusing by some.

### Add options

Knowing all flags in eu4 is no small feat!
Thus, it is also a daunting task, which may make this boring for some since most nations are once you've never heard of.
This calls for an options menu which lets you constrain the flags you get.
Some examples include:

* Constrain by geography i.e Continent, Subcontinent, Region.
* Constrain by year, most reasonably nations which are present in 1444.
* Constrain by size e.g only show nations with more than one province.
* Exclude formables.
* Exclude easter eggs.

### Expand database

The current database makes implementing the aforementioned constraints quite difficult.
So I basically need to scrape the eu4 wiki for more data.
