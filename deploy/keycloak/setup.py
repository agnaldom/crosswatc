#!/usr/bin/env python3
"""Create local-only Keycloak credentials once; never overwrite existing secrets."""
import os
from pathlib import Path
import secrets

root = Path(__file__).resolve().parents[2]
path = root / ".env.keycloak"
if path.exists():
    print(".env.keycloak already exists; credentials preserved.")
else:
    entries = {
        "KC_BOOTSTRAP_ADMIN_USERNAME": "keycloak-admin",
        "KC_BOOTSTRAP_ADMIN_PASSWORD": secrets.token_urlsafe(32),
        "CROSSWATC_OIDC_CLIENT_SECRET": secrets.token_urlsafe(32),
        "CROSSWATC_LOCAL_USER_PASSWORD": secrets.token_urlsafe(24),
    }
    with os.fdopen(os.open(path, os.O_WRONLY | os.O_CREAT | os.O_EXCL, 0o600), "w") as stream:
        stream.write("# Local development only. Generated credentials; do not commit.\n")
        stream.writelines(f"{key}={value}\n" for key, value in entries.items())
    print("Created .env.keycloak (mode 0600). Local user: crosswatc-dev.")
