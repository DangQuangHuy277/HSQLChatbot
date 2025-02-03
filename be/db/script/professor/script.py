import os
import re

import psycopg2
import requests
from bs4 import BeautifulSoup
from dotenv import load_dotenv

load_dotenv()
DATABASE_URL = os.getenv('DATABASE_URL')


# Function to save data to the database
def save_to_database(data):
    conn = None
    cursor = None
    try:
        conn = psycopg2.connect(DATABASE_URL)
        cursor = conn.cursor()

        # Insert
        faculty_set = set(professor["faculty"] for professor in data)
        faculty_map = {}  # Initialize an empty dictionary to store faculty mappings

        for faculty in faculty_set:
            cursor.execute(
                """
                INSERT INTO faculty (name, type) VALUES (%s, %s)
                ON CONFLICT (name) DO UPDATE SET name = EXCLUDED.name
                RETURNING id, name;
                """,
                (faculty, extract_faculty_type(faculty)),
            )
            result = cursor.fetchone()  # Fetch the result of the current query

            if result:
                faculty_id, faculty_name = result
                faculty_map[faculty_name] = faculty_id

        # Insert professor data
        for professor in data:
            cursor.execute(
                """
                INSERT INTO professor (name, academic_rank, degree, faculty_id) VALUES (%s, %s, %s, %s)
                ON CONFLICT (name) DO NOTHING;
                """,
                (professor["name"], professor["academic_rank"], professor["degree"], faculty_map.get(professor["faculty"])),
            )

        conn.commit()
        print("Data saved to database successfully.")

    except Exception as e:
        print("Error saving to database:", e)

    finally:
        if cursor:
            cursor.close()
        if conn:
            conn.close()


# Function to scrape data from the website
def scrape_uet_staff():
    url = "https://uet.vnu.edu.vn/doi-ngu-can-bo/"
    response = requests.get(url, verify=False)

    if response.status_code != 200:
        print("Failed to retrieve the webpage.")
        return

    soup = BeautifulSoup(response.content, "html.parser")

    # Find all department sections
    sections = soup.find_all("table")

    data = []
    for section in sections[:-1]: # Exclude the last section which contains Adjunct Professors
        for row in section.find('tbody').find_all('tr'):
            if not row.find('td').text.isdigit():
                continue

            academic_rank, degree, name = extract_academic_rank_and_degree(row.find_all('td')[1].text)
            professor_data = {
                "name": name,
                "academic_rank": academic_rank,
                "degree": degree,
                "faculty": row.find_all('td')[2].text.strip()
            }
            data.append(professor_data)

    save_to_database(data)


def extract_faculty_type(faculty_name):
    pattern = r"^(Khoa|Viện|Trung tâm nghiên cứu)\b"  # Match leading type keywords
    match = re.match(pattern, faculty_name.strip())  # Use strip() to remove leading/trailing spaces
    if match:
        return match.group(1)  # Return the type of faculty (e.g., Khoa, Viện, Trung tâm)
    return None  # Return None if no match is foun


def extract_academic_rank_and_degree(full_name):
    # Define regex to capture academic rank, degree, and name
    pattern = r"^(?:(GS|PGS)\.?)?\s?(?:(TSKH|TS|ThS)\.?)?\s?(.*)$"
    match = re.match(pattern, full_name)

    if match:
        academic_rank = match.group(1)
        degree = match.group(2)
        name = match.group(3)

        # Map the academic rank and degree to Vietnamese values
        academic_rank_map = {"GS": "Giáo sư", "PGS": "Phó giáo sư"}
        degree_map = {"TSKH": "Tiến sĩ khoa học", "TS": "Tiến sĩ", "ThS": "Thạc sĩ"}

        academic_rank = academic_rank_map.get(academic_rank, None)
        degree = degree_map.get(degree, None)

        return academic_rank, degree, name.strip()
    return None, None, full_name


# Run the script
if __name__ == "__main__":
    scrape_uet_staff()
