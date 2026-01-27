# GitHub Secrets Setup Guide

To complete the plugin signing workflow, you need to add secrets to your GitHub repository.

## Required Secret

**Secret Name**: `GRAFANA_ACCESS_POLICY_TOKEN`

## Steps to Add the Secret

### 1. Create Grafana Cloud Access Policy Token

1. Go to [Grafana Cloud](https://grafana.com/signup) and sign in
2. Navigate to: **My Account** > **Security** > **Access Policies**
3. Click **"Create access policy"**
4. Configure:
   - **Realm**: `your-org-name (all-stacks)`
   - **Scope**: `plugins:write`
5. Click **"Create token"**
6. **Save the token** - you'll need it in the next step

### 2. Add Secret to GitHub Repository

1. Go to your GitHub repository: `https://github.com/jaops-space/grafana-yamcs-jaops`
2. Navigate to: **Settings** > **Secrets and variables** > **Actions**
3. Click **"New repository secret"**
4. Set:
   - **Name**: `GRAFANA_ACCESS_POLICY_TOKEN`
   - **Secret**: Paste your Grafana Cloud access policy token
5. Click **"Add secret"**

### 3. Test the Workflow

Once the secret is added:
1. Create a version tag: `npm version patch`
2. Push the tag: `git push origin main --follow-tags`
3. Check the GitHub Actions workflow runs successfully
4. Verify the plugin is signed in the release artifacts

## Security Notes

- Keep your access policy token secure
- Don't share or commit the token in code
- Consider setting an expiration date for the token
- Rotate tokens periodically for security

## Troubleshooting

If the workflow fails:
- Check the secret name matches exactly: `GRAFANA_ACCESS_POLICY_TOKEN`
- Verify the token has `plugins:write` scope
- Ensure your Grafana Cloud account has the necessary permissions
