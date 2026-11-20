# US-020: Look up a tire by code

As a mechanic, I want to look up a tire by its code, so that I see which vehicle it belongs to.

## Acceptance criteria

### AC-20.1: Find a tire
- Given a tire with code T-1 exists
- When I search for T-1
- Then I see its vehicle and number

## Out of scope
- editing tires

## External dependencies
- backend-repository: needs a tire lookup endpoint
  Suggested: /story:add As a frontend, I want GET /tires?code=… so that the SPA can show a tire
  Assumed contract (mock): GET /tires?code=… → { code, vehicle, number, category }

  ```json
  { "code": "T-1", "vehicle": "truck", "number": 3, "category": "A" }
  ```
- infra: needs a staging bucket
  - Given nothing, When read, Then ignored
