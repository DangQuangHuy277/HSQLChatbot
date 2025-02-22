import os
from datetime import datetime

import psycopg2
import pdfplumber
from dotenv import load_dotenv

load_dotenv()

DATABASE_URL = os.getenv('DATABASE_URL')

def extract_data_from_pdf(pdf_path):
    """Extract student data from the PDF using extract_table for tabular pages and extract_text otherwise."""
    data = []  # List to store extracted data

    with pdfplumber.open(pdf_path) as pdf:
        program_name = None
        for page in pdf.pages:
            # Attempt to extract table data
            table = page.extract_table()

            if table:
                # Process the table rows
                for row in table[1:]:  # Skip the header row
                    if not row or not row[0] or not row[0].strip()[0].isdigit():
                        continue  # Skip rows without valid student data

                    code = row[1].replace(' ', '')
                    name = row[2].strip()
                    birthday = row[3].strip()
                    gender = row[4].strip()
                    administrative_class_name = row[-1].strip()

                    student_data = {
                        'code': code,
                        'name': name,
                        'birthday': birthday,
                        'gender': gender,
                        'administrative_class_name': administrative_class_name,
                        'program_abbreviation': administrative_class_name.rsplit('/', 1)[-1].removeprefix('CQ-').strip(' 0123456789'),
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

    # retrieve mapping from program name to program id
    cursor.execute("""
        SELECT id, abbreviation FROM program;
    """)

    program_mapping = {abb.lower().strip(): id for id, abb in cursor.fetchall() if abb is not None}

    # Iterate over the extracted student data
    for student in data:
        # Convert the birthday string to a date object
        try:
            birthday = datetime.strptime(student['birthday'], "%d/%m/%Y").date()
        except ValueError:
            birthday = None  # Handle any invalid dates gracefully

        upsert_administrative_class(cursor, student, program_mapping)

# Insert student data into the 'student' table
        cursor.execute("""
            INSERT INTO student (code, name, gender, birthday, email, administrative_class_id)
            VALUES (%s, %s, %s, %s, %s, %s)
        """, (
            student['code'],
            student['name'],
            student['gender'],
            birthday,
            student.get('email', student["code"] + "@vnu.edu.vn"),  # Optional email field
            student['administrative_class_id']
        ))

    # Commit the transaction
    conn.commit()

    # Close the cursor and connection
    cursor.close()
    conn.close()

    print(f"Successfully saved {len(data)} students to the database.")


def upsert_administrative_class(cursor, student, program_mapping):
    # Check if the administrative class already exists in the database
    cursor.execute("""
            INSERT INTO administrative_class (name, program_id)
            VALUES (%s, %s)
            ON CONFLICT (name) 
            DO UPDATE SET name = administrative_class.name  -- No-op update, just to trigger the RETURNING
            RETURNING id
        """, (student['administrative_class_name'],program_mapping.get(student['program_abbreviation'].lower().strip(), None),))
    administrative_class_id = cursor.fetchone()
    # Reset max id of the administrative_class table
    cursor.execute("""
            SELECT setval('administrative_class_id_seq', MAX(id), true) FROM administrative_class;
        """)
    student['administrative_class_id'] = administrative_class_id
    student['email'] = student['code'] + "@vnu.edu.vn"


if __name__ == "__main__":
    pdf_directory = "/home/huy/Code/Personal/KLTN/be/db/script/student/source/"  # Replace with the actual directory path

    for filename in os.listdir(pdf_directory):
        pdf_path = None
        data = None
        # filename = "/home/huy/Code/Personal/KLTN/be/db/script/student/source_backup/DSSV-k69.pdf"
        if filename.endswith(".pdf"):
            pdf_path = os.path.join(pdf_directory, filename)
        if pdf_path:
            data = extract_data_from_pdf(pdf_path)

        if data:
            save_student_data(data)
        else:
            print(f"No data extracted from {filename}.")
