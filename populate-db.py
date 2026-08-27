#!/bin/python

import sqlite3
from urllib.request import urlretrieve
import urllib

# CREATE TABLE Countries
# (name varchar(255) PRIMARY KEY,
# flag BLOB,
# tag varchar(255),
# capital_subcontinent varchar(255),
# capital_region varchar(255),
# capital_province varchar(255),
# notes varchar(255));

SQLITE_INSERT_QUERY = "INSERT INTO Countries (tag, name, flag, capital_subcontinent, capital_region, capital_province, notes) VALUES (?, ?, ?, ?, ?, ?, ?)"
FLAG_PATH = "./flags/"
WIKI_LINK = "https://eu4.paradoxwikis.com/"
FLAG_WIDTHS = [1152, 1000, 900, 875, 840, 821, 800, 768, 350, 192]

class Country:
    idx: int
    name: str
    tag: str
    subcontinent: str
    region: str
    province: str
    notes: str

    def __init__(self, idx: int, name: str, tag: str, subcontinent: str, region: str, province: str, notes: str):

        self.idx = idx
        self.name = name
        self.tag = tag
        self.subcontinent = subcontinent
        self.region = region
        self.province = province
        self.notes = notes

    def __str__(self):
        return(f"Country:\n\t{self.idx}\n\t{self.name}\n\t{self.tag}\n\t{self.subcontinent}\n\t{self.region}\n\t{self.province}\n\t{self.notes}")
    

def extract_line_info(line) -> Country:
    line = line.replace("</td>\n", "")
    entries = line.split("<td>")[1:]

    # Index
    idx = int(entries[0])
    # Name
    name = entries[1].replace("</a></b>", "").split("\">")[-1]
    # Tag
    tag = entries[2]
    # Capital info
    subcontinent = "-"
    region = "-"
    province = "-"
    if entries[3] != "–":
        temp = entries[3].split(" / ")
        subcontinent = temp[0]
        region = temp[1]
        province = temp[2].split(" ")[0]
    # Notes
    notes = entries[4]

    return Country(idx, name, tag, subcontinent, region, province, notes)


conn = sqlite3.connect('countries_test.db')
cursor = conn.cursor()

with open("eu4-countries-table.html", "r") as raw:
    text = raw.read()
    lines = text.split("</tr>\n<tr>")

    for l in lines:
        c: Country = extract_line_info(l)
        print(f"Adding data for {c.name}")
        img_file_path = FLAG_PATH + c.name + ".png"
        found_image = False
        for width in FLAG_WIDTHS:
            img_link = WIKI_LINK + "thumb.php?f=" + urllib.parse.quote_plus(c.name) + f".png&width={width-1}"
            path, headers = urlretrieve(img_link, img_file_path)
            for h in headers.items():
                if (h[1] == "image/png"):
                    found_image = True
                    break
            if found_image:
                break
            if width == FLAG_WIDTHS[-1]:
                print(f"Could not find image for {c.name}! Aborting...")
                conn.close()
                exit(1)

        with open(img_file_path, "rb") as img:
            img_data = img.read()
            data_tuple = (c.tag, c.name, img_data, c.subcontinent, c.region, c.province, c.notes)
            cursor.execute(SQLITE_INSERT_QUERY, data_tuple)

conn.commit()
conn.close()
