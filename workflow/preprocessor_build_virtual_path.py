import csv
import os
import sys
import json

SYMLINK_FOLDER = "/data"
NEWPATH_DATA = {}
MANIFEST_ROOTS = []

def create_sym_link(source_path, target_path):
    user_path = (os.path.expanduser('~'))
    symlink_folder = "/data"
    path_parts = source_path.split("/")
    filename = path_parts[len(path_parts) - 1]
    try:
        target_path = f"{user_path}{symlink_folder}/{target_path}/{filename}"
        print(target_path)

        folder_path = os.path.dirname(target_path)
        if os.path.exists(folder_path) == False:
            os.makedirs(folder_path)

        os.symlink(source_path, target_path)
        print(f"Created symlink from {source_path} to {target_path}")
    except FileExistsError as e:
        print(f"Symlink already exists for {target_path}")
    except FileNotFoundError as e:
        print("File Error")
        print(e)
    except Exception as e:
        print("Unexpected Error")
        print(e)


def build_container_csv_paths():
    for root, dirs, files in os.walk(SYMLINK_FOLDER, topdown=True):
        for file in files:
            new_path = f"{root}/{file}"
            NEWPATH_DATA[new_path] = new_path

def get_manifest_roots():
    f = open('/job/workflow/work_order.json')
    work_order = json.load(f)
    return work_order['ManifestRoots']

def main(csv_file):
    build_container_csv_paths()
    dir_roots = get_manifest_roots()
    print(NEWPATH_DATA)
    with open(csv_file, 'r') as file:
        reader = csv.DictReader(file)
        headers = reader.fieldnames

        if 'source_path' not in headers or 'target_path' not in headers:
            print("Required headers not found in CSV file.")
            return

        for row in reader:

            # replace user machine path with container path
            for root in dir_roots:
                source_path = row['source_path'].replace(root, SYMLINK_FOLDER)

            target_path = row['target_path']
            create_sym_link(source_path, target_path)


if __name__ == "__main__":
    if len(sys.argv) != 2:
        print(f"Usage: {sys.argv[0]} <csv_file>")
        sys.exit(1)

    csv_file = sys.argv[1]
    main(sys.argv[1])
