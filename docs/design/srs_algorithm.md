# SRS Algorithm Parameters and Design

This document outlines the specific parameters and design decisions for the Spaced Repetition System (SRS) algorithm used in the Scry API. The implementation is based on the SM-2 algorithm with some modifications to better suit the application's needs.

## Algorithm Overview

The SRS algorithm determines when a card should be reviewed next based on the user's performance. The core components include:

1. **Interval calculation** - Determines the number of days until the next review
2. **Ease factor adjustment** - Modifies the difficulty rating of a card
3. **Review scheduling** - Sets the exact date/time for the next review

The algorithm takes into account:
- The current interval
- The current ease factor
- The user's performance on the current review (outcome)
- The number of consecutive correct reviews

## Default Parameters

### Core Limits

- **Minimum Ease Factor:** 1.3
  - Cards cannot become more difficult than this threshold
  - Prevents cards from becoming impossible to learn

- **Maximum Ease Factor:** 2.5
  - Cards cannot become easier than this threshold
  - Prevents review intervals from growing too quickly

### Ease Factor Adjustments

Adjustments made to the ease factor based on review outcome:

| Outcome | Adjustment | Explanation |
|---------|------------|-------------|
| Again   | -0.20      | Significant decrease for failed cards |
| Hard    | -0.15      | Modest decrease for difficult cards |
| Good    | 0.0        | No change for expected performance |
| Easy    | +0.15      | Modest increase for easy cards |

### Interval Modifiers

Modifiers applied to the interval calculation:

| Outcome | Modifier | Explanation |
|---------|----------|-------------|
| Again   | 0.0      | Reset interval completely |
| Hard    | 1.2      | Small increase for difficult cards |
| Good    | 1.0      | Use ease factor directly |
| Easy    | 1.3      | Additional multiplier beyond ease factor |

### First Review Intervals

Special case handling for the first review of a card (or after an "Again" outcome):

| Outcome | Interval (days) | Explanation |
|---------|----------------|-------------|
| Hard    | 1              | Review tomorrow |
| Good    | 1              | Review tomorrow |
| Easy    | 2              | Skip a day |

### Special Timing

- **"Again" Review Minutes:** 10
  - Failed cards will be shown again after a 10-minute delay
  - Allows for quick reinforcement of difficult material

## Algorithm Examples

### Example 1: New Card Flow

Starting with a new card with default parameters:
- Initial interval: 0 days
- Initial ease factor: 2.5
- Initial consecutive correct: 0

**First review (outcome: "Good")**:
- New ease factor: 2.5 (unchanged)
- New interval: 1 day (first review interval)
- Consecutive correct: 1
- Next review: Tomorrow

**Second review (outcome: "Good")**:
- New ease factor: 2.5 (unchanged)
- New interval: 2.5 days (1 day * 2.5 ease factor)
- Consecutive correct: 2
- Next review: 2-3 days later

**Third review (outcome: "Easy")**:
- New ease factor: 2.5 (already at maximum)
- New interval: 8 days (2.5 days * 2.5 ease factor * 1.3 modifier)
- Consecutive correct: 3
- Next review: 8 days later

### Example 2: Lapse Handling

Starting with a card with:
- Current interval: 15 days
- Current ease factor: 2.3
- Current consecutive correct: 5

**Review (outcome: "Again")**:
- New ease factor: 2.1 (2.3 - 0.2)
- New interval: 0 days (reset)
- Consecutive correct: 0 (reset)
- Next review: 10 minutes later

**Next review (outcome: "Good")**:
- New ease factor: 2.1 (unchanged)
- New interval: 1 day (first review interval)
- Consecutive correct: 1
- Next review: Tomorrow

**Next review (outcome: "Good")**:
- New ease factor: 2.1 (unchanged)
- New interval: 2.1 days (1 day * 2.1 ease factor)
- Consecutive correct: 2
- Next review: 2 days later

### Example 3: Hard Cards

Starting with a card with:
- Current interval: 5 days
- Current ease factor: 2.0
- Current consecutive correct: 3

**Review (outcome: "Hard")**:
- New ease factor: 1.85 (2.0 - 0.15)
- New interval: 6 days (5 days * 1.2 modifier)
- Consecutive correct: 4
- Next review: 6 days later

**Next review (outcome: "Hard")**:
- New ease factor: 1.7 (1.85 - 0.15)
- New interval: 7.2 days (6 days * 1.2 modifier)
- Consecutive correct: 5
- Next review: 7 days later

## Design Decisions and Rationale

### SM-2 Modifications

1. **First Review Intervals**
   - The original SM-2 algorithm has a fixed interval of 1 day for first review
   - Our modification allows different first intervals based on outcome
   - Rationale: Allows easier cards to be spaced further apart immediately

2. **"Again" Review Delay**
   - Standard SM-2 doesn't specify precise timing for failed cards
   - Our modification adds a 10-minute delay
   - Rationale: Short delay provides a quick reinforcement opportunity

3. **Lapse Handling**
   - Standard SM-2 reduces intervals to 1 day upon failure
   - Our modification resets to 0 days (10 minutes)
   - Rationale: Provides more immediate reinforcement for failed material

4. **Maximum Ease Factor**
   - Standard SM-2 allows unlimited ease factor growth
   - Our modification caps at 2.5
   - Rationale: Prevents intervals from growing too quickly, which can lead to over-confidence

5. **Different Interval Modifiers**
   - Added specific modifiers for each outcome
   - Rationale: More granular control over spacing algorithm

### Immutable Implementation

The algorithm is implemented with immutability in mind:
- Functions operate on input data and return new objects
- No direct mutation of the original UserCardStats objects
- Explicit time handling for better testability

### Service Design

The SRS algorithm is provided through a service interface:
- Allows for multiple implementations
- Supports dependency injection
- Separates pure calculation from storage concerns

## Performance Considerations

The algorithm is designed to be lightweight and efficient:
- Pure functions have minimal overhead
- No database or external service calls in core calculation
- Creating new UserCardStats objects has negligible performance impact

## Future Enhancements

Potential future enhancements to consider:
1. **Learning Stage** - Add a separate initial learning phase with shorter intervals
2. **Time-of-Day Scheduling** - Consider user's preferred study times
3. **Forgetting Curve Adaptation** - Dynamically adjust parameters based on user performance
4. **Difficulty Buckets** - Allow categorization of cards into difficulty groups
5. **Review Limit Management** - Strategies for managing daily review load
