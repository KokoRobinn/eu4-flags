#!/bin/python

import os
import sqlite3
import urllib.parse

from playwright.sync_api import sync_playwright

SQLITE_INSERT_QUERY = """
INSERT INTO Countries
(id, tag, name, flag_path, capital_subcontinent, capital_region, capital_province, notes)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
"""

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

    def __init__(
        self,
        idx: int,
        name: str,
        tag: str,
        subcontinent: str,
        region: str,
        province: str,
        notes: str,
    ):
        self.idx = idx
        self.name = name
        self.tag = tag
        self.subcontinent = subcontinent
        self.region = region
        self.province = province
        self.notes = notes
    
    def __str__(self):
        return (
            f"Country:\n"
            f"\t{self.idx}\n"
            f"\t{self.name}\n"
            f"\t{self.tag}\n"
            f"\t{self.subcontinent}\n"
            f"\t{self.region}\n"
            f"\t{self.province}\n"
            f"\t{self.notes}"
        )

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
    
    return Country(
        idx,
        name,
        tag,
        subcontinent,
        region,
        province,
        notes,
    )

def download_flag(page, country_name: str, output_path: str) -> bool:
    """
    Download a country's flag using the browser context.
    
    ```
    Returns True if an image was successfully downloaded.
    """
    
    for width in FLAG_WIDTHS:
        img_link = (
            WIKI_LINK
            + "thumb.php?f="
            + urllib.parse.quote_plus(country_name)
            + f".png&width={width - 1}"
        )
    
        print(f"  Trying width {width - 1}...")
    
        try:
            response = page.request.get(img_link)
    
            content_type = response.headers.get("content-type", "")
    
            if response.ok and content_type.startswith("image/"):
                with open(output_path, "wb") as file:
                    file.write(response.body())
    
                print(f"  Downloaded flag ({width - 1}px)")
                return True
    
            print(
                f"  Failed: HTTP {response.status} "
                f"({content_type})"
            )
    
        except Exception as e:
            print(f"  Request failed: {e}")
    
    return False

def main():
    os.makedirs(FLAG_PATH, exist_ok=True)
    
    conn = sqlite3.connect("countries_test.db")
    cursor = conn.cursor()
    
    try:
        # Read country table
        with open("eu4-countries-table.html", "r") as raw:
            text = raw.read()
    
        lines = text.split("</tr>\n<tr>")
    
        with sync_playwright() as p:
            print("Starting Chromium...")
    
            browser = p.chromium.launch(
                headless=True
            )
    
            context = browser.new_context(
                user_agent=(
                    "Mozilla/5.0 (Windows NT 10.0; Win64; x64) "
                    "AppleWebKit/537.36 (KHTML, like Gecko) "
                    "Chrome/131.0.0.0 Safari/537.36"
                )
            )
    
            page = context.new_page()
    
            print("Opening wiki...")
    
            # Open the actual site first so that the browser can
            # complete any client-side challenge and establish
            # the required cookies/session.
            page.goto(
                WIKI_LINK,
                wait_until="domcontentloaded",
                timeout=60000,
            )
    
            # Give client-side challenge scripts time to complete.
            page.wait_for_timeout(3000)
    
            print("Wiki loaded.")
            print()
    
            for l in lines:
                c: Country = extract_line_info(l)
    
                print(f"Adding data for {c.name}")
    
                img_path = os.path.join(
                    FLAG_PATH,
                    c.name + ".png",
                )
    
                if c.name == "Mamluks":
                    success = download_flag(
                        page,
                        "The Mamluks",
                        img_path,
                    )
                else:
                    success = download_flag(
                        page,
                        c.name,
                        img_path,
                    )
    
                if not success:
                    print(
                        f"Could not find image for {c.name}! "
                        "Aborting..."
                    )
    
                    conn.close()
                    browser.close()
                    return
    
                data_tuple = (
                    c.idx,
                    c.tag,
                    c.name,
                    img_path,
                    c.subcontinent,
                    c.region,
                    c.province,
                    c.notes,
                )
    
                cursor.execute(
                    SQLITE_INSERT_QUERY,
                    data_tuple,
                )
    
            browser.close()
    
        conn.commit()
    
    finally:
        conn.close()

if __name__ == "__main__":
    main()

