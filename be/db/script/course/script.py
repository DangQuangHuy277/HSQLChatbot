import os

import psycopg2
import requests
from bs4 import BeautifulSoup
from dotenv import load_dotenv

program_course = {
    1: "https://uet.vnu.edu.vn/chuong-trinh-dao-tao-nganh-cong-nghe-thong-tin-10/",
    2: "https://uet.vnu.edu.vn/chuong-trinh-dao-tao-nganh-ky-thuat-may-tinh-10/",
    3: "https://uet.vnu.edu.vn/chuong-trinh-dao-tao-nganh-vat-ly-ky-thuat-10/",
    4: "https://uet.vnu.edu.vn/chuong-trinh-dao-tao-nganh-co-ky-thuat-24/",
    5: "https://uet.vnu.edu.vn/chuong-trinh-dao-tao-nganh-cong-nghe-ky-thuat-xay-dung-10/",
    6: "https://uet.vnu.edu.vn/chuong-trinh-dao-tao-nganh-cong-nghe-ky-thuat-co-dien-tu-clc-tt23-3/",
    7: "https://uet.vnu.edu.vn/chuong-trinh-dao-tao-nganh-cong-nghe-hang-khong-vu-tru-9/",
    8: "https://uet.vnu.edu.vn/chuong-trinh-dao-tao-nganh-khoa-hoc-may-tinh-21/",
    9: "https://uet.vnu.edu.vn/chuong-trinh-dao-tao-nganh-cong-nghe-ky-thuat-dien-tu-vien-thong-clc-tt23-5/",
    10: "https://uet.vnu.edu.vn/chuong-trinh-dao-tao-nganh-cong-nghe-nong-nghiep-7/",
    11: "https://uet.vnu.edu.vn/chuong-trinh-dao-tao-nganh-ky-thuat-dieu-khien-va-tu-dong-hoa-7/",
    12: "https://uet.vnu.edu.vn/chuong-trinh-dao-tao-nganh-tri-tue-nhan-tao-6/",
    13: "https://uet.vnu.edu.vn/chuong-trinh-dao-tao-nganh-ky-thuat-nang-luong-10/",
    14: "https://uet.vnu.edu.vn/chuong-trinh-dao-tao-nganh-thong-thong-tin-21/",
    15: "https://uet.vnu.edu.vn/chuong-trinh-dao-tao-nganh-mang-may-tinh-va-truyen-thong-du-lieu-13/",
    16: "https://uet.vnu.edu.vn/chuong-trinh-dao-tao-nganh-cong-nghe-thong-tin-dinh-huong-thi-truong-nhat-ban-10/",
    17: "https://uet.vnu.edu.vn/chuong-trinh-dao-tao-nganh-ky-thuat-robot-7/",
    18: "https://uet.vnu.edu.vn/chuong-trinh-dao-tao-nganh-thiet-ke-cong-nghiep-va-hoa-3/",
    # 3 new programs doesn't have course list yet
}

load_dotenv()
DATABASE_URL = os.getenv('DATABASE_URL')


def save_courses_to_database(courses, program_id):
    with psycopg2.connect(DATABASE_URL) as conn:
        with conn.cursor() as cur:
            for course in courses:
                cur.execute("""
                    INSERT INTO course (code, name, english_name, credits, practice_hours, theory_hours, self_learn_hours)
                    VALUES (%s, %s, %s, %s, %s, %s, %s)
                    ON CONFLICT (code) DO UPDATE SET 
                        name = EXCLUDED.name,
                        english_name = EXCLUDED.english_name,
                        credits = EXCLUDED.credits,
                        practice_hours = EXCLUDED.practice_hours,
                        theory_hours = EXCLUDED.theory_hours,
                        self_learn_hours = EXCLUDED.self_learn_hours
                    RETURNING id;
                """, (
                    course['code'], course['name'], course['english_name'], course['credits'],
                    course['practice_hours'], course['theory_hours'], course['self_learn_hours']
                ))
                course_id = cur.fetchone()[0]

                # Reset id so that it is continuous
                cur.execute("""SELECT setval('course_id_seq', MAX(id), true) FROM course;""")

                # Link course to program
                cur.execute("""
                    INSERT INTO course_program (program_id, course_id)
                    VALUES (%s, %s)
                    ON CONFLICT DO NOTHING;
                """, (program_id, course_id))


def update_prerequisites(prerequisite_updates):
    with psycopg2.connect(DATABASE_URL) as conn:
        with conn.cursor() as cur:
            for course_code, prereq_code in prerequisite_updates:
                if not prereq_code:
                    continue
                cur.execute("""
                    UPDATE course c1
                    SET prerequisite = c2.id
                    FROM course c2
                    WHERE c1.code = %s AND c2.code = %s;
                """, (course_code, prereq_code))

def standardize_course_name(name: str) -> str:
    # Remove special marks
    name = name.replace('(*)', '').strip()

    name = name.replace('(bắt buộc)', '').strip()
    # Remove multiple spaces
    name = ' '.join(name.split())
    return name

def crawl_and_store(program_id, url):
    response = requests.get(url, verify=False)
    response.raise_for_status()
    soup = BeautifulSoup(response.content, "html.parser")
    course_rows = soup.select("table tr")

    prerequisite_updates = []
    courses = []

    for row in course_rows[1:]:
        columns = row.find_all("td")
        if len(columns) < 8 or not columns[1].text.strip() or ' ' in columns[1].text.strip():
            continue


        name_parts = columns[2].text.strip().split('\n')
        course = {
            'code': columns[1].text.strip(),
            'name': name_parts[0],
            'english_name': name_parts[1] if len(name_parts) > 1 else None,
            'credits': int(columns[3].text.strip()),
            'practice_hours': int(columns[4].text.strip()) if columns[4].text.strip() else 0,
            'theory_hours': int(columns[5].text.strip()) if columns[5].text.strip() else 0,
            'self_learn_hours': int(columns[6].text.strip()) if columns[6].text.strip() else 0,
        }
        courses.append(course)

        if prereq := columns[7].text.strip():
            prerequisite_updates.append((course['code'], prereq))

    save_courses_to_database(courses, program_id)
    update_prerequisites(prerequisite_updates)


if __name__ == "__main__":
    for program_id, url in program_course.items():
        crawl_and_store(program_id, url)
    print("Data has been successfully crawled and stored.")
