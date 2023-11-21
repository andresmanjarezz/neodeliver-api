# validator used to check input on API call:
https://github.com/go-playground/validator

# Neodeliver mailing lists
Neodeliver has his own user account to manage his contacts exactly the same way as other users
The organization id of neodeliver is available through the environment variable: `NEODELIVER_ORGANIZATION_ID`

# possible graphql function params:
- graphql.ResolveParams
- rbac.RBAC
- args struct { ... }

## Auth0 Credential env variables
The system is connecting to the `Auth0` for authentication, password change..etc. You have to configure the following env variables in `.env` 
- `AUTH0_TENANT` : Auth0 domain url
- `AUTH0_MANAGEMENT_CLIENT_ID` : Management API client ID
- `AUTH0_MANAGEMENT_CLIENT_SECRET` : Management API client Secret
- `AUTH0_MANAGEMENT_AUDIENCE` : Management API Audience
- `AUTH0_MANAGEMENT_CONNECTION` : Auth0 data connectin name

