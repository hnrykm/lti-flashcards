#!/usr/bin/env sh
# Makes the local Moodle (docker compose) a working LTI 1.3 platform for this tool:
# registers the tool, creates the BIO101 course with a student and a Flashcards
# activity, then saves Moodle's issuer/client/deployment in the app's platforms table.
# Safe to run more than once. Run from the repo root after `docker compose up -d`.
set -eu

out=$(docker compose exec -T -e TOOL_URL="${TOOL_URL:-http://localhost:3000}" moodle php < scripts/moodle-setup.php | tail -n 1)
field() { printf '%s' "$out" | sed -n "s/.*\"$1\":\"\([^\"]*\)\".*/\1/p" | sed 's#\\/#/#g'; }

issuer=$(field issuer)
client_id=$(field client_id)
deployment_id=$(field deployment_id)
auth_url=$(field auth_login_url)
jwks_url=$(field jwks_url)
activity=$(field activity_url)

if [ -z "$client_id" ]; then
  echo "Moodle setup failed:" >&2
  echo "$out" >&2
  exit 1
fi

docker compose exec -T db psql -q -v ON_ERROR_STOP=1 -U flashcards -d flashcards <<SQL
INSERT INTO platforms (issuer, client_id, deployment_id, auth_login_url, jwks_url)
VALUES ('$issuer', '$client_id', '$deployment_id', '$auth_url', '$jwks_url')
ON CONFLICT (issuer, client_id) DO UPDATE
SET deployment_id = EXCLUDED.deployment_id,
    auth_login_url = EXCLUDED.auth_login_url,
    jwks_url = EXCLUDED.jwks_url;
SQL

cat <<EOF
Moodle is set up as an LTI 1.3 platform.
  issuer         $issuer
  client_id      $client_id
  deployment_id  $deployment_id

Launch: log in at $issuer (admin / Admin123!, or student / Student123!)
and open $activity
EOF
