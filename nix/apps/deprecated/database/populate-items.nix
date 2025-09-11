{
  inputs,
  writers,
  python3Packages,
  ...
}:
writers.writePython3Bin "populate-items" {
  libraries = [python3Packages.requests];
  doCheck = false;
}
''
  import json
  import sqlite3
  import sys

  DB_PATH = "./web/public/database.sqlite3"
  WOWSIMS_DB_JSON = "${inputs.wowsims}/assets/database/db.json"

  def load_wowsims_items():
      # load items, gems, and enchants from wowsims db
      print(f"Loading items from {WOWSIMS_DB_JSON}")

      with open(WOWSIMS_DB_JSON, 'r') as f:
          db_data = json.load(f)

      # Combine items, gems, and enchants into one collection
      all_items = []

      if "items" in db_data:
          all_items.extend(db_data["items"])
          print(f"Found {len(db_data['items'])} items")

      if "gems" in db_data:
          all_items.extend(db_data["gems"])
          print(f"Found {len(db_data['gems'])} gems")

      if "enchants" in db_data:
          all_items.extend(db_data["enchants"])
          print(f"Found {len(db_data['enchants'])} enchants")

      print(f"Total: {len(all_items)} items/gems/enchants in wowsims database")

      return all_items

  def populate_items_table():
      # populate items table with wowsims data
      if not os.path.exists(DB_PATH):
          print(f"Error: Database not found at {DB_PATH}")
          sys.exit(1)

      items_data = load_wowsims_items()

      conn = sqlite3.connect(DB_PATH)
      cursor = conn.cursor()

      try:
          print("Clearing existing items...")
          cursor.execute("DELETE FROM items")

          print("Inserting items...")
          insert_count = 0

          # process all items (items, gems, enchants combined)
          for item in items_data:
              item_id = item.get("id")

              if not item_id:
                  continue

              name = item.get("name", "")
              icon = item.get("icon", "")
              quality = item.get("quality", 0)
              item_type = item.get("type", 0)

              # store stats as json string for flexibility
              stats_json = json.dumps(item.get("stats", {}))

              cursor.execute("""
                  INSERT OR REPLACE INTO items
                  (id, name, icon, quality, type, stats)
                  VALUES (?, ?, ?, ?, ?, ?)
              """, (item_id, name, icon, quality, item_type, stats_json))

              insert_count += 1

              if insert_count % 1000 == 0:
                  print(f"Processed {insert_count} items...")

          conn.commit()
          print(f"✓ Successfully inserted {insert_count} items")

      except Exception as e:
          print(f"Error populating items: {e}")
          conn.rollback()
          sys.exit(1)
      finally:
          conn.close()

  if __name__ == "__main__":
      import os
      populate_items_table()
''
