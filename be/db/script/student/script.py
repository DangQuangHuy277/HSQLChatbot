import os
import re
from datetime import datetime

import psycopg2
from PyPDF2 import PdfReader
from dotenv import load_dotenv

load_dotenv()

DATABASE_URL = os.getenv('DATABASE_URL')

def extract_data_from_pdf(pdf_path):
    """Extract student data from the PDF."""
    reader = PdfReader(pdf_path)
    data = []  # List to store extracted data

    for page in reader.pages:
        text = page.extract_text()
        for line in text.split("\n"):
            if not line[0].isdigit():
                continue  # Skip lines that don't start with a digit (i.e., don't look like student data)

            elements = [x for x in line.split(' ')[1:] if x]  # Remove empty strings

            # Extract the code (first element)
            code = elements[0]

            # Find the valid date (birthday)
            birthday, birthday_index = next(((x, i) for i, x in enumerate(elements) if is_valid_date(x)), (None, None))
            if not birthday:
                continue  # Skip the line if no valid birthday is found
            name = ' '.join(elements[1:birthday_index])
            gender = elements[birthday_index + 1]
            administrative_class_name = ' '.join(elements[birthday_index + 2:])
            student_data = {
                'code': code,
                'name': name,
                'birthday': birthday,
                'gender': gender,
                'administrative_class_name': administrative_class_name
            }
            data.append(student_data)

    return data

def is_valid_date(date_string, date_format="%d/%m/%Y"):
    try:
        # Try parsing the date string
        datetime.strptime(date_string, date_format)
        return True
    except ValueError:
        return False


def save_student_data(data):
    # Connect to the PostgreSQL database using the connection string
    conn = psycopg2.connect(DATABASE_URL)

    cursor = conn.cursor()

    # Iterate over the extracted student data
    for student in data:
        # Convert the birthday string to a date object
        try:
            birthday = datetime.strptime(student['birthday'], "%d/%m/%Y").date()
        except ValueError:
            birthday = None  # Handle any invalid dates gracefully


        # Check if the administrative class already exists in the database
        cursor.execute("""
            INSERT INTO administrative_class (name)
            VALUES (%s)
            ON CONFLICT (name) 
            DO UPDATE SET name = administrative_class.name  -- No-op update, just to trigger the RETURNING
            RETURNING id
        """, (student['administrative_class_name'],))

        administrative_class_id = cursor.fetchone()

        # Reset max id of the administrative_class table
        cursor.execute("""
            SELECT setval('administrative_class_id_seq', MAX(id), true) FROM administrative_class;
        """)

        student['administrative_class_id'] = administrative_class_id
        student['email'] = student['code'] + "@vnu.edu.vn"

        # Insert student data into the 'student' table
        cursor.execute("""
            INSERT INTO student (code, name, gender, birthday, email, program_id, administrative_class_id)
            VALUES (%s, %s, %s, %s, %s, %s, %s)
        """, (
            student['code'],
            student['name'],
            student['gender'],
            birthday,
            student.get('email', student["code"] + "@vnu.edu.vn"),  # Optional email field
            student.get('program_id', None),
            student['administrative_class_id']
        ))

    # Commit the transaction
    conn.commit()

    # Close the cursor and connection
    cursor.close()
    conn.close()

    print(f"Successfully saved {len(data)} students to the database.")


if __name__ == "__main__":
    pdf_directory = "/home/huy/Code/Personal/KLTN/be/db/script/student/source/"  # Replace with the actual directory path

    for filename in os.listdir(pdf_directory):
        if filename.endswith(".pdf"):
            pdf_path = os.path.join(pdf_directory, filename)
    data = extract_data_from_pdf(pdf_path)

    if data:
        save_student_data(data)
    else:
        print(f"No data extracted from {filename}.")
