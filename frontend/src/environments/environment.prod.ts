// 各値は GitHub Actions の Secrets で注入される（.github/workflows/deploy-frontend.yml 参照）
// 必要な Secrets: API_BASE_URL, AUTH0_DOMAIN, AUTH0_CLIENT_ID, AUTH0_AUDIENCE
export const environment = {
  production: true,
  apiBaseUrl: '__API_BASE_URL__',
  auth0: {
    domain: '__AUTH0_DOMAIN__',
    clientId: '__AUTH0_CLIENT_ID__',
    audience: '__AUTH0_AUDIENCE__',
  },
};
