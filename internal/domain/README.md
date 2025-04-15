# Domain Models

This document describes the core domain models of the Scry API application, their relationships, and their purpose.

## Models Overview

### User

The `User` model represents a registered user of the application. It contains:
- Unique identifier (`ID`)
- Authentication details (`Email`, `HashedPassword`)
- Timestamps for creation and updates

Users can create Memos and review Cards that are generated from those Memos.

### Memo

The `Memo` model represents a text entry submitted by a user for generating flashcards. It contains:
- Unique identifier (`ID`)
- Reference to the user who created it (`UserID`)
- The actual memo text (`Text`)
- The processing status (`Status`)
- Timestamps for creation and updates

Memos can be in one of five states:
- `pending`: The memo has been submitted but not yet processed
- `processing`: The memo is currently being processed to generate cards
- `completed`: The memo has been successfully processed and all cards have been generated
- `completed_with_errors`: The memo was processed but some cards failed to be generated
- `failed`: The memo processing failed completely

### Card

The `Card` model represents a flashcard generated from a user's memo. It contains:
- Unique identifier (`ID`)
- References to the user (`UserID`) and memo (`MemoID`) it was generated from
- The content of the card (`Content`) as a flexible JSON structure
- Timestamps for creation and updates

Card content is stored as a JSON structure to allow for flexibility in the card format. The standard format includes:
- `front`: The question or prompt side of the card
- `back`: The answer side of the card
- `hint` (optional): A hint to help the user remember the answer
- `tags` (optional): Keywords or categories associated with the card
- `image_url` (optional): URL to an image associated with the card

### UserCardStats

The `UserCardStats` model tracks a user's spaced repetition statistics for a specific card. It contains:
- Composite key of `UserID` and `CardID`
- Spaced repetition algorithm parameters (`Interval`, `EaseFactor`, `ConsecutiveCorrect`)
- Review timing information (`LastReviewedAt`, `NextReviewAt`)
- Review statistics (`ReviewCount`)
- Timestamps for creation and updates

UserCardStats implements a modified version of the SM-2 spaced repetition algorithm to determine when cards should be reviewed based on the user's past performance.

## Relationships

The domain models have the following relationships:

1. **User-Memo**: One-to-many. A user can create multiple memos.
2. **Memo-Card**: One-to-many. A memo can generate multiple cards.
3. **User-Card**: One-to-many. A user owns multiple cards (generated from their memos).
4. **User-Card-Stats**: Many-to-many with attributes. A user has statistics for each of their cards, stored in the UserCardStats model.

## Domain Logic

The domain models encapsulate the following core business logic:

1. **User authentication**: The User model supports email-based authentication.
2. **Memo processing workflow**: The Memo model tracks the state of processing memos to generate cards.
3. **Spaced repetition scheduling**: The UserCardStats model implements the SM-2 algorithm to schedule card reviews.
4. **Card content management**: The Card model provides flexible storage for card content while maintaining references to the source memo.

Each model includes validation logic to ensure data integrity, and methods to support the required business operations (e.g., updating review statistics, changing memo status, etc.).
