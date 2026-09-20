# Deploy checklist

1. **Secrets** – copy `.env.example` → `.env`; set `DB_PASSWORD` and a random `JWT_ACCESS_SECRET` (`openssl rand -base64 48`), `APP_ENV=production`.
2. **First admin** – set `ADMIN_EMAIL`/`ADMIN_PASSWORD`, or leave the password empty: a random one is printed once on the first start
   (stderr) and must be changed at first login.
3. **HTTPS** – behind a TLS reverse proxy set `TRUSTED_PROXIES=<proxy ip>` and keep `COOKIE_SECURE=true`.
   On a plain-HTTP internal network use `COOKIE_SECURE=false`.
4. **Offline server** – nothing is fetched at runtime (fonts are bundled). Only the *build* needs internet; build the image elsewhere.
   Keep the server clock correct (NTP or manual) – token expiry depends on it.
5. **Do not publish** PostgreSQL's port; only the app port should be reachable.
6. **Verify** – `ADMIN_PASSWORD='...' ./scripts/smoke_test.sh http://localhost:8080`
7. **Backup** – e.g. `pg_dump -Fc -U postgres fits_db > fits_$(date +%F).dump` from cron; copy it off the server.
