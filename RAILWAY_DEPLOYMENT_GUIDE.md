# Railway Deployment Guide for mana-wise

## Prerequisites

1. Railway.app account ([Sign up](https://railway.app))
2. Git repository connected to Railway
3. MongoDB Atlas database (or Railway-hosted MongoDB plugin)
4. Environment variables configured

## Deployment Steps

### 1. Create/Connect GitHub Repository

```bash
# If not already done, push to GitHub
git remote add origin https://github.com/YOUR_USERNAME/mana-wise.git
git branch -M main
git push -u origin main
```

### 2. Create Railway Project

1. Log in to [Railway Dashboard](https://railway.app/dashboard)
2. Click **New Project**
3. Select **Deploy from GitHub repo**
4. Select `mana-wise` repository
5. Authorize Railway to access your repo

### 3. Configure Environment Variables

In Railway Dashboard, go to **Project Settings** → **Variables** and add:

```env
# Database
MONGODB_URI=mongodb+srv://USER:PASSWORD@cluster.mongodb.net/manawise

# Security
JWT_SECRET=your-secure-random-secret-here
ENVIRONMENT=production

# API Configuration
PORT=8080
CORS_ORIGINS=https://yourdomain.com,https://www.yourdomain.com

# Scryfall API
SCRYFALL_RATE_LIMIT=10

# Impact Analysis
EDH_IMPACT_CACHE_TTL_HOURS=24
EDH_IMPACT_WEIGHTS_PRICE=0.4
EDH_IMPACT_WEIGHTS_EDHREC=0.5
EDH_IMPACT_WEIGHTS_REPRINT=0.1

# Logging
LOG_LEVEL=info
```

### 4. Configure MongoDB (if using Railway plugin)

1. In Railway Project, click **New** → **Add Service**
2. Select **MongoDB**
3. Railway auto-configures `MONGODB_URI`
4. Approve and Railway redeploys automatically

### 5. Configure Health Check

Railway automatically detects the Dockerfile and health check. Verify in **Settings**:

- **Build Command**: (Auto-detected from Dockerfile)
- **Start Command**: (Auto-detected)
- **Port**: `8080`
- **Health Check Path**: `/api/v1/health`

### 6. Deploy

```bash
# Railway auto-deploys on git push to main
git commit -m "chore: prepare for Railway deployment"
git push origin main

# Monitor deployment:
# - Railway Dashboard → Deployments tab
# - Watch build logs in real-time
```

### 7. Verify Deployment

Once deployment is complete:

```bash
# Check health endpoint
curl https://your-railway-domain.railway.app/api/v1/health

# Test score endpoint
curl -X POST https://your-railway-domain.railway.app/api/v1/score \
  -H "Content-Type: application/json" \
  -d '{
    "decklist": [...],
    "format": "commander"
  }'
```

## Troubleshooting

### Build Fails: "go build ./cmd/server not found"
- **Solution**: Ensure `cmd/server/main.go` exists and project root has `go.mod`

### "MONGODB_URI not found"
- **Solution**: Add MongoDB plugin or set `MONGODB_URI` environment variable in Railway

### Health check fails (503 Unavailable)
- **Solution**: Verify `GET /api/v1/health` returns 200 in local testing
  ```bash
  go run ./cmd/server &
  curl http://localhost:8080/api/v1/health
  ```

### Port binding error
- **Solution**: Railway sets `PORT` env var automatically; ensure Go server reads it:
  ```go
  port := os.Getenv("PORT")
  if port == "" {
    port = "8080"
  }
  http.ListenAndServe(":"+port, handler)
  ```

## Frontend Deployment

The Dockerfile includes a multi-stage build for the React frontend:

1. **Stage 1** (web-builder): `npm ci` + `npm run build` → `/web/dist`
2. **Stage 2** (builder): Go binary build
3. **Stage 3** (runtime): Alpine Linux with both binary + static files

The Go server serves the frontend at `/` (ensure router includes static file handler).

## Database Backups

For production, configure MongoDB backups:

1. In MongoDB Atlas: **Backup & Restore** → Enable automated backups
2. Set retention to 30 days minimum
3. Test restoration process monthly

## Monitoring & Logs

1. Railway Dashboard → **Logs** tab
2. Set up log alerting for errors:
   - Filter: `level=error`
   - Notify on threshold: 5 errors/hour

## Rolling Back Deployment

```bash
# If deployment is broken, Railway allows rollback
# Via Dashboard: Deployments → Select previous build → Redeploy

# Or reset git and push old commit
git revert HEAD
git push origin main
```

## CI/CD Pipeline

Railway auto-deploys on every push to `main`. For staging environment:

1. Create `staging` branch
2. Point to staging MongoDB instance
3. Disable auto-deploy on staging in Railway settings

## Cost Optimization

- **Compute**: $5/mo for basic tier (suitable for low-traffic APIs)
- **Database**: MongoDB Atlas free tier (512MB) or Railway's $11/mo tier
- **Bandwidth**: $0.10/GB outbound

For cost tracking, use Railway's **Billings** dashboard.

## Next Steps

1. ✅ Push to GitHub
2. ✅ Connect to Railway
3. ✅ Configure environment variables
4. ✅ Deploy
5. ✅ Monitor logs and metrics
6. ✅ Set up error alerting

---

**Support**: Railway docs: https://docs.railway.app
