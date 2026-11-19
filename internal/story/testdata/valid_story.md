# US-012: Cancel a booking

As a club player, I want to cancel, so that the court is freed.

## Acceptance criteria

### AC-12.1: Cancel own booking
- Given I have a booking
- When I cancel it
- Then the court is free

### AC-12.2: Cannot cancel late
- Given a booking starts in 1h
- When I cancel
- Then I see an error

## Out of scope
- refunds
