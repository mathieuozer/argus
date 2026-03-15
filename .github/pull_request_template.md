## Summary

<!-- Brief description of what this PR does and why -->

## Changes

<!-- List the key changes made in this PR -->

-

## Test Plan

<!-- How were these changes tested? -->

- [ ] Unit tests pass (`make test`)
- [ ] Integration tests pass (`make test-int`)
- [ ] Manual testing performed (describe below)

## Checklist

- [ ] Code follows the project conventions defined in CLAUDE.md
- [ ] All new DB tables include `tenant_id` with RLS policy
- [ ] New API endpoints have cross-tenant access tests (must return 403)
- [ ] No secrets or credentials committed
- [ ] No raw agent inputs/outputs logged (Tier 3 data protection)
- [ ] Protobuf definitions updated if gRPC interfaces changed (`make proto`)
- [ ] Documentation updated if needed
- [ ] Compliance: changes reviewed for GDPR / NIS2 / KVKK / FedRAMP impact
