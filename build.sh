#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PLUGIN_FILE="$ROOT_DIR/sml-wp-plugin.php"
PLUGIN_DIR_NAME="$(basename "$ROOT_DIR")"
RELEASE_KIND="${1:-patch}"

if [[ ! -f "$PLUGIN_FILE" ]]; then
  echo "Fehler: $PLUGIN_FILE nicht gefunden."
  exit 1
fi

if ! command -v zip >/dev/null 2>&1; then
  echo "Fehler: 'zip' ist nicht installiert."
  exit 1
fi

current_version="$(sed -nE 's/^[[:space:]]*\*[[:space:]]*Version:[[:space:]]*([0-9]+\.[0-9]+\.[0-9]+)[[:space:]]*$/\1/p' "$PLUGIN_FILE" | head -n1)"
if [[ -z "$current_version" ]]; then
  echo "Fehler: Konnte aktuelle Version in $PLUGIN_FILE nicht lesen."
  exit 1
fi

IFS='.' read -r major minor patch <<< "$current_version"

case "$RELEASE_KIND" in
  major)
    major=$((major + 1))
    minor=0
    patch=0
    ;;
  minor)
    minor=$((minor + 1))
    patch=0
    ;;
  patch)
    patch=$((patch + 1))
    ;;
  *)
    if [[ "$RELEASE_KIND" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
      IFS='.' read -r major minor patch <<< "$RELEASE_KIND"
    else
      echo "Usage: ./build.sh [patch|minor|major|X.Y.Z]"
      exit 1
    fi
    ;;
esac

new_version="${major}.${minor}.${patch}"

# Update plugin header version (first matching "Version:" line).
tmp_file="$(mktemp)"
awk -v v="$new_version" '
  BEGIN { done = 0 }
  {
    if (!done && $0 ~ /^[[:space:]]*\*[[:space:]]*Version:[[:space:]]*[0-9]+\.[0-9]+\.[0-9]+[[:space:]]*$/) {
      sub(/[0-9]+\.[0-9]+\.[0-9]+/, v)
      done = 1
    }
    print
  }
' "$PLUGIN_FILE" > "$tmp_file"
mv "$tmp_file" "$PLUGIN_FILE"

# Keep asset versions in sync for cache busting.
sed -i '' -E "/wp_enqueue_style\\('sml-admin'/ s/'[0-9]+\\.[0-9]+\\.[0-9]+'/'${new_version}'/" "$PLUGIN_FILE"
sed -i '' -E "/wp_enqueue_script\\('sml-admin'/ s/'[0-9]+\\.[0-9]+\\.[0-9]+'/'${new_version}'/" "$PLUGIN_FILE"
sed -i '' -E "/wp_enqueue_style\\('sml-frontend'/ s/'[0-9]+\\.[0-9]+\\.[0-9]+'/'${new_version}'/" "$PLUGIN_FILE"

zip_versioned="$ROOT_DIR/${PLUGIN_DIR_NAME}-${new_version}.zip"
zip_latest="$ROOT_DIR/${PLUGIN_DIR_NAME}.zip"
rm -f "$zip_versioned" "$zip_latest"

(
  cd "$ROOT_DIR/.."
  zip -r "$zip_versioned" "$PLUGIN_DIR_NAME" \
    -x "${PLUGIN_DIR_NAME}/.git/*" \
       "${PLUGIN_DIR_NAME}/.DS_Store" \
       "${PLUGIN_DIR_NAME}/spec.md" \
       "${PLUGIN_DIR_NAME}/*.zip"
)

cp "$zip_versioned" "$zip_latest"

echo "Version: ${current_version} -> ${new_version}"
echo "ZIP: $zip_versioned"
echo "Latest: $zip_latest"
