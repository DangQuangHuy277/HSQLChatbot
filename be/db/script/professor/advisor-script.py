import os

import pdfplumber
import requests
from bs4 import BeautifulSoup
from dotenv import load_dotenv
import psycopg2

load_dotenv()
DATABASE_URL = os.getenv('DATABASE_URL')

k_class_map = {}
class_map = {}

def save_advisor_data(class_data):
    conn = None
    cursor = None
    try:
        conn = psycopg2.connect(DATABASE_URL)
        cursor = conn.cursor()

        for class_entry in class_data:
            class_name = class_entry["class_name"]
            advisor_name = class_entry["advisor_name"]

            # Fetch advisor ID from professor table
            cursor.execute(
                """
                SELECT id FROM professor WHERE name = %s;
                """,
                (advisor_name,)
            )
            result = cursor.fetchone()

            if result:
                advisor_id = result[0]
            else:
                # Insert new professor into professor table
                cursor.execute(
                    """
                    INSERT INTO professor (name) VALUES (%s)
                    RETURNING id;
                    """,
                    (advisor_name,)
                )
                advisor_id = cursor.fetchone()[0]
                print(f"Inserted new advisor '{advisor_name}' with ID {advisor_id}.")

            # Update advisor_id in administrative_class
            cursor.execute(
                """
                UPDATE administrative_class
                SET advisor_id = %s
                WHERE name = %s;
                """,
                (advisor_id, class_name)
            )
            print(f"Updated class '{class_name}' with advisor '{advisor_name}' (ID: {advisor_id}).")

        conn.commit()
        print("Advisor data saved successfully.")

    except Exception as e:
        print("Error saving advisor data:", e)

    finally:
        if cursor:
            cursor.close()
        if conn:
            conn.close()



def get_advisor_name_and_class(advisor_info):
    # Find the content inside parentheses
    start = advisor_info.find('(') + 1
    end = advisor_info.find(')')
    if 0 < start < end:
        content_inside_parentheses = advisor_info[start:end]
        # Split the content into advisor name and class name using '_'
        parts = content_inside_parentheses.split('_')
        if len(parts) == 2:
            advisor_name = parts[0].strip()
            class_name = parts[1].strip()
            return advisor_name, class_name
    return None, None

def standardize_class_name(class_name):
    return class_map.get(k_class_map.get(class_name))



# Function to scrape advisor data for a specific program (e.g., K66, K67)
def scrape_advisors_for_year(school_year, base_url):
    page = 1
    class_data = []

    while True:
        print(f"Scraping page {page} for {school_year}...")
        params = {'page': page}
        response = requests.get(base_url, params=params, verify=False)  # Use verify=False for SSL issues

        if response.status_code != 200:
            print(f"Failed to fetch page {page}. Status code: {response.status_code}")
            break

        soup = BeautifulSoup(response.content, "html.parser")
        courses = soup.select('div.coursebox')

        if not courses:
            print(f"No more courses found for {school_year}.")
            break

        for course in courses:
            advisor_info = course.select_one('h3.coursename a').text.strip()
            advisor_name, class_name = get_advisor_name_and_class(advisor_info)

            if advisor_name and class_name:
                class_data.append({"class_name": standardize_class_name(class_name), "advisor_name": advisor_name})
        page += 1

    save_advisor_data(class_data)


# Main function to scrape advisors for all programs
def scrape_all_programs():
    years = {
        "K64": "https://courses.uet.edu.vn/course/index.php?categoryid=57",
        "K65": "https://courses.uet.edu.vn/course/index.php?categoryid=58",
        "K66": "https://courses.uet.edu.vn/course/index.php?categoryid=71",
        "K67": "https://courses.uet.edu.vn/course/index.php?categoryid=77",
    }


    with pdfplumber.open("/home/huy/Code/Personal/KLTN/be/db/script/professor/source/name-conversion.pdf") as pdf:
        for page in pdf.pages:
            text = page.extract_text()
            lines = text.split('\n')
            for line in lines:
                parts = line.split()
                if not parts[0].isdigit():
                    continue
                class_map[parts[1]] = parts[2]

    for key, value in class_map.items():
        year = int(key[3:7])
        k = year - 1955
        class_name_suffix = ''.join(key[13:].split('-'))
        k_class_map['K' + str(k) + class_name_suffix] = key


    for school_year, base_url in years.items():
        print(f"Starting scrape for {school_year}...")
        scrape_advisors_for_year(school_year, base_url)


# Run the script
if __name__ == "__main__":
    scrape_all_programs()
