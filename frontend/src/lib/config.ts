// Centralized frontend environment configuration
export const config = {
  apiBaseUrl: import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080/api/v1',
  keycloakUrl: import.meta.env.VITE_KEYCLOAK_URL || 'http://localhost:8081',
  keycloakRealm: import.meta.env.VITE_KEYCLOAK_REALM || 'sentinel',
  keycloakClientId: import.meta.env.VITE_KEYCLOAK_CLIENT_ID || 'sentinel-frontend',
}
