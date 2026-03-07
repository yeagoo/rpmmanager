#!/usr/bin/env bash
set -euo pipefail

# Arguments:
#   $1 = repo base URL (e.g., http://repo)
#   $2 = product name (e.g., testapp)
#   $3 = el version (e.g., el9)
#   $4 = arch (e.g., x86_64)
#   $5 = expected version (e.g., 0.1.0)

REPO_BASE="$1"
PRODUCT="$2"
EL_VERSION="$3"
ARCH="$4"
EXPECTED_VERSION="$5"

echo "=== RPM Installation Test ==="
echo "Product: $PRODUCT"
echo "EL: $EL_VERSION"
echo "Arch: $ARCH"
echo "Expected version: $EXPECTED_VERSION"
echo ""

# Step 1: Import the GPG key
echo "Importing GPG key..."
rpm --import "$REPO_BASE/$PRODUCT/gpg.key"
echo "GPG key imported."

# Step 2: Create yum repo config
echo "Configuring yum repository..."
cat > /etc/yum.repos.d/${PRODUCT}.repo << EOF
[${PRODUCT}]
name=${PRODUCT}
baseurl=${REPO_BASE}/${PRODUCT}/${EL_VERSION}/${ARCH}/
enabled=1
gpgcheck=1
repo_gpgcheck=0
gpgkey=${REPO_BASE}/${PRODUCT}/gpg.key
EOF

echo "Repo config:"
cat /etc/yum.repos.d/${PRODUCT}.repo
echo ""

# Step 3: Install the package
echo "Installing $PRODUCT..."
dnf install -y "$PRODUCT"
echo "Package installed."

# Step 4: Verify the binary exists
echo "Checking binary..."
if ! command -v "$PRODUCT" &> /dev/null; then
    echo "ERROR: $PRODUCT binary not found in PATH"
    exit 1
fi
echo "Binary found: $(which $PRODUCT)"

# Step 5: Run the binary and check output
echo "Running $PRODUCT..."
output=$("$PRODUCT" 2>&1 || true)
echo "Output: $output"

if echo "$output" | grep -q "$EXPECTED_VERSION"; then
    echo "SUCCESS: Version $EXPECTED_VERSION confirmed"
else
    echo "ERROR: Expected version $EXPECTED_VERSION not found in output"
    exit 1
fi

# Step 6: Verify RPM metadata
echo "Checking RPM info..."
rpm -qi "$PRODUCT" | head -10
echo ""

echo "=== All checks passed ==="
