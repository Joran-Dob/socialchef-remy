# Agent Rules for SocialChef Remy

## API Development

### Bruno Collection Updates (REQUIRED)

When adding or modifying API endpoints, **ALWAYS** update the Bruno collection in `/bruno/`:

1. **New Endpoints**: Create a new `.bru` file in the appropriate folder
2. **Modified Endpoints**: Update existing `.bru` files
3. **Update README**: Add new endpoints to the endpoints table in `bruno/README.md`

#### Folder Structure
- `0-Health/` - Health checks (no auth)
- `1-Recipe/` - Recipe import endpoints
- `2-Embedding/` - Embedding generation
- `3-Search/` - Search endpoints  
- `4-Bulk-Import/` - Bulk import endpoints

#### Bruno File Template
```
meta {
  name: Endpoint Name
  type: http
  seq: 1
}

post {
  url: {{baseUrl}}/api/endpoint
  body: json
  auth: inherit
}

body:json {
  {
    "field": "value"
  }
}

script:post-response {
  test("Status is 200", function() {
    expect(res.status).to.equal(200);
  });
  
  // Store vars for chaining
  bru.setVar("varName", res.body.id);
}
```

#### Checklist for API Changes
- [ ] `.bru` file created/updated
- [ ] Tests added in `script:post-response`
- [ ] Variables stored with `bru.setVar()` for chaining
- [ ] `bruno/README.md` endpoints table updated
- [ ] Collection structure section updated (if new folder)

## Database Migrations (REQUIRED)

When modifying database schema, **ALWAYS** create migrations in BOTH locations:

1. **`/Users/jorandob/Documents/Projects/SocialChef/socialchef-remy/supabase/migrations/`** - For the Remy service
2. **`/Users/jorandob/Documents/Projects/SocialChef/socialchef-supabase/supabase/migrations/`** - For the main Supabase project

### Migration Naming Convention
```
YYYYMMDD_description.sql
```

Examples:
- `20260310_add_bulk_import_jobs.sql`
- `20260311_add_user_preferences.sql`

### Checklist for Schema Changes
- [ ] Migration file created in `socialchef-remy/supabase/migrations/`
- [ ] Migration file copied to `socialchef-supabase/supabase/migrations/`
- [ ] Both files have identical content
- [ ] Migration is idempotent (uses `IF NOT EXISTS`, `IF EXISTS`)
- [ ] Indexes added for new tables/columns
- [ ] Foreign key constraints defined with `ON DELETE` behavior
