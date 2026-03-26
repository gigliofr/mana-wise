# Quick Start: Deploy to Railway in 5 Minutes

## Prerequisites
- ✅ GitHub account with repo pushed
- ✅ Railway.app account (free tier available)
- ✅ MongoDB Atlas cluster (free tier 512MB available)

---

## Step 1: Connect GitHub to Railway (2 min)

1. Go to https://railway.app/dashboard
2. Click **New Project**
3. Select **Deploy from GitHub repo**
4. Authorize & select `mana-wise` repository
5. Railway auto-detects `Dockerfile` and starts building ✅

---

## Step 2: Configure Environment Variables (2 min)

In Railway Dashboard → **Variables** tab, add:

### Required
```
MONGODB_URI=mongodb+srv://user:pass@cluster.mongodb.net/manawise
JWT_SECRET=your-secure-random-string-here
```

### Optional (defaults provided)
```
PORT=8080
ENVIRONMENT=production
LOG_LEVEL=info
EDH_IMPACT_CACHE_TTL_HOURS=24
EDH_IMPACT_WEIGHTS_PRICE=0.4
EDH_IMPACT_WEIGHTS_EDHREC=0.5
EDH_IMPACT_WEIGHTS_REPRINT=0.1
```

### Get MongoDB Connection String
1. Go to https://www.mongodb.com/cloud/atlas
2. Create free cluster (512MB)
3. Click **Connect** → **Connection String**
4. Copy and replace in Railway variables

---

## Step 3: Deploy (1 min)

1. Railway should already be building
2. Watch **Deployments** tab for build progress
3. Once green ✅, deployment is live

---

## Step 4: Verify Deployment (30 sec)

Test health endpoint:
```bash
curl https://your-railway-domain.railway.app/api/v1/health
# Expected: 200 OK with {"status": "ok"}
```

Test score endpoint:
```bash
curl -X POST https://your-railway-domain.railway.app/api/v1/score \
  -H "Content-Type: application/json" \
  -d '{
    "decklist": [
      {
        "id": "test-card-1",
        "name": "Test Land",
        "cmc": 0,
        "type_line": "Basic Land",
        "edhrec_rank": 100
      }
    ],
    "format": "commander"
  }'
# Expected: 200 OK with score_detail object
```

---

## Troubleshooting

### "Build failed: Cannot find go.mod"
- ✅ Already fixed: `go.mod` exists in repo root

### "Health check fails (503)"
- Verify `GET /api/v1/health` endpoint exists (it does ✅)
- Wait 30s for first deployment to stabilize

### "MONGODB_URI not set"
- Check Railway **Variables** tab
- Ensure connection string is correct format
- Test locally: `mongodb+srv://user:pass@cluster.mongodb.net/db`

### "Port binding error"
- Railway sets `PORT` env var automatically ✅
- Go server reads it: `os.Getenv("PORT")`

---

## What Gets Deployed

✅ Go binary (manawise server)  
✅ React frontend (compiled to static files)  
✅ Health check endpoint  
✅ POST /score API endpoint  
✅ All dependencies from go.mod  

---

## After Deployment

1. **Monitor Logs**: Railway Dashboard → **Logs** tab
2. **Set Custom Domain**: Railway Dashboard → **Settings** → **Domains**
3. **Enable Auto-Deploy**: Push to `main` branch = auto-redeploy
4. **Backups**: Configure MongoDB backups (Atlas dashboard)

---

## Next Steps

- ✅ Code is in GitHub
- ✅ Dockerfile is production-ready
- ✅ Environment variables are documented
- 🚀 **Just connect to Railway and deploy!**

---

**Time to Deploy**: ~5 minutes  
**Time to First Request**: ~30 seconds  
**Cost**: Free tier (first 5GB, then $5/mo compute + DB costs)

**Go to**: https://railway.app/dashboard
