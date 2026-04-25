# YouTube API Credentials Setup Guide

## Overview

This guide walks you through obtaining a YouTube Data API key for the SocialChef Remy recipe extractor. The YouTube API enables Remy to fetch video metadata and captions from YouTube recipe videos, expanding beyond Instagram and TikTok content sources.

## Prerequisites

Before starting, you'll need:

- A Google account (Gmail or Google Workspace)
- Access to the Google Cloud Console

## Step-by-Step Setup Instructions

### Step 1: Access Google Cloud Console

1. Open your browser and go to the [Google Cloud Console](https://console.cloud.google.com/)
2. Sign in with your Google account if you aren't already

### Step 2: Create or Select a Project

1. At the top of the page, click the project dropdown (it may say "Select a project")
2. To create a new project:
   - Click **"New Project"**
   - Enter a project name (e.g., "SocialChef-Remy")
   - Select an organization (if applicable) or leave as "No organization"
   - Click **"Create"**
3. Wait a few seconds for the project to be created, then select it from the dropdown

### Step 3: Enable YouTube Data API v3

1. In the left navigation menu, go to **"APIs & Services" > "Library"**
2. In the search box, type **"YouTube Data API v3"**
3. Click on **"YouTube Data API v3"** in the results
4. Click the **"Enable"** button
5. Wait for the API to be enabled (this may take a minute)

### Step 4: Create API Credentials

1. Once enabled, click **"Create Credentials"** or go to **"APIs & Services" > "Credentials"**
2. Click **"Create Credentials"** and select **"API key"**
3. Google will generate and display your API key immediately
4. Copy this key to a secure location. You'll need it for your environment variables.

### Step 5: Secure Your API Key (Recommended)

1. Next to your newly created key, click **"Restrict Key"**
2. Under **"Application restrictions"**:
   - Choose **"HTTP referrers (web sites)"** if using from a web app
   - Choose **"IP addresses"** if calling from a specific server
   - Or leave as **"None"** for local development (not recommended for production)
3. Under **"API restrictions"**:
   - Select **"Restrict key"**
   - Find and select **"YouTube Data API v3"**
4. Click **"Save"**

### Step 6: Add to Environment Variables

Add the following to your `.env` file:

```bash
YOUTUBE_API_KEY=your_api_key_here
```

## API Key vs OAuth 2.0

This project uses an **API Key**, not OAuth 2.0. Here's the difference:

| Authentication Method | Use Case | What Remy Uses |
| :--- | :--- | :--- |
| **API Key** | Read-only access to public data (video metadata, captions) | ✅ Yes |
| **OAuth 2.0** | Access to user data (uploading videos, accessing private playlists) | ❌ No |

Remy only needs to read public video information. It does not upload content or access user accounts. Therefore, a simple API key is sufficient and simpler to manage than OAuth credentials.

## Quota Limits

The YouTube Data API has quota limits that reset daily at midnight Pacific Time.

### Default Quota

| Resource | Default Quota |
| :--- | :--- |
| **Daily quota** | 10,000 units per day |
| **Video search** | 100 units per request |
| **Video details** | 1 unit per request |
| **Captions list** | 50 units per request |
| **Captions download** | 200 units per request |

### Example Usage Calculation

For a typical recipe video fetch (metadata + captions):
- Video details: 1 unit
- Captions download: 200 units
- **Total per video**: ~201 units

With the default 10,000 unit quota, you can process approximately **49 videos per day**.

### Monitoring Usage

1. Go to [Google Cloud Console](https://console.cloud.google.com/)
2. Navigate to **"APIs & Services" > "Dashboard"**
3. Click on **"YouTube Data API v3"**
4. View quota usage charts and current day's consumption

### Requesting Higher Quotas

If you need more than 10,000 units per day:

1. Go to **"APIs & Services" > "Quotas"**
2. Find the quota you want to increase
3. Click the edit icon and **"Apply for higher quota"**
4. Fill out the application with your use case details
5. Google's review typically takes 2-3 business days

## Troubleshooting

### Error: "API key not valid"

**Cause**: The key was entered incorrectly or hasn't propagated yet.

**Solutions**:
- Double-check you copied the entire key without extra spaces
- Wait 5-10 minutes after creation for the key to activate
- Verify the key is enabled in the Credentials section

### Error: "YouTube Data API has not been used in project"

**Cause**: The API wasn't enabled for this project.

**Solutions**:
- Go to **"APIs & Services" > "Library"**
- Search for "YouTube Data API v3"
- Click **"Enable"**

### Error: "The request cannot be completed because you have exceeded your quota"

**Cause**: You've reached the daily quota limit.

**Solutions**:
- Wait until midnight Pacific Time for the quota to reset
- Check your usage in the Cloud Console dashboard
- Consider applying for a quota increase if this happens regularly
- Review your application for unnecessary API calls

### Error: "Video unavailable" or "This video does not exist"

**Cause**: The video is private, deleted, or region-restricted.

**Solutions**:
- Verify the video URL is correct and publicly accessible
- Check if the video plays in a logged-out browser session
- Some videos have embedding disabled, which may affect caption access

### Error: "Captions not available"

**Cause**: The video doesn't have captions or they're disabled.

**Solutions**:
- Not all YouTube videos have captions
- Auto-generated captions may not be available for all languages
- The video owner may have disabled caption downloads

### Error: "The API key is restricted"

**Cause**: You've set application or API restrictions that are being violated.

**Solutions**:
- Go to **"APIs & Services" > "Credentials"**
- Edit your API key
- Check that your application's IP or referrer is allowed
- Verify "YouTube Data API v3" is in the allowed APIs list

## Additional Resources

- [YouTube Data API Overview](https://developers.google.com/youtube/v3)
- [YouTube Data API Quota Calculator](https://developers.google.com/youtube/v3/determine_quota_cost)
- [Google Cloud Console Help](https://cloud.google.com/support)
- [YouTube API Error Documentation](https://developers.google.com/youtube/v3/docs/errors)

## Next Steps

Once you have your API key configured, return to the [main README](../README.md) to continue setup or refer to the scraper documentation for implementation details.
