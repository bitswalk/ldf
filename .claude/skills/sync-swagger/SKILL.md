---
name: sync-swagger
description: Regenerate the OpenAPI/Swagger specification from swag annotations and validate all API handlers are documented. Use after modifying API endpoints.

---

# Sync Swagger

Regenerate the OpenAPI spec and validate annotation coverage.

## Steps

### 1. Regenerate the spec

```bash
~/go/bin/swag init --dir src/ldfd,src/common -g docs.go -o src/ldfd/docs --parseDependency --parseInternal
```

### 2. Validate coverage

Check that all handler methods in `src/ldfd/api/` have Swagger annotations. Every exported `Handle*` method should have at minimum:
- `@Summary`
- `@Description`
- `@Tags`
- `@Produce json`
- `@Router` with correct path and method
- `@Success` with response type
- `@Failure` for error cases (at least 500)
- `@Security BearerAuth` for protected endpoints

Search for handlers missing annotations:

```bash
grep -rn "func (h \*Handler) Handle" src/ldfd/api/ --include="*.go" | while read line; do
  file=$(echo "$line" | cut -d: -f1)
  linenum=$(echo "$line" | cut -d: -f2)
  prev=$((linenum - 1))
  if ! sed -n "${prev}p" "$file" | grep -q "@Router"; then
    echo "MISSING ANNOTATION: $line"
  fi
done
```

### 3. Verify build

```bash
task build:srv
```

### 4. Report

Show a summary of:
- Total endpoints documented
- Any handlers missing annotations
- Any warnings from swag init
