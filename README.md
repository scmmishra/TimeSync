# TimeSync (working name)

TimeSync is a macOS menu-bar app for remote teams that shows the **current local time of teammates**, automatically and reliably, without manual timezone management.

The product exists to answer a single question with confidence: *“What time is it for them right now?”*

Nothing more, nothing less.

## The problem

In distributed teams, people travel constantly. Static “home time zones” become inaccurate within days, and most tools rely on users to remember to update their location or timezone manually. This leads to friction, unnecessary interruptions, and poor coordination.

TimeSync treats timezone as a **live signal**, not a profile field.

## The idea

Every user who opts in runs TimeSync on their Mac. Their device reports its own timezone automatically and periodically, without requiring any action from them. Other team members see the resulting local time in their menu bar, always up to date.

There is no location tracking, no guessing, and no manual maintenance.

## What the product does

TimeSync lives entirely in the macOS menu bar. When opened, it shows a list of team members who are currently sharing their timezone, sorted by **time difference from the viewer**.

For each visible teammate, the app shows:

- Their name and avatar
- Their exact current local time
- A friendly country label (with a flag anchored to the avatar)
- A subtle UI hint if they are outside their configured working hours (for example, a moon icon)

The list is intentionally simple and uncluttered. There is no search, filtering, scheduling, or meeting planning in v1.

## Teams and membership

TimeSync is team-based. There is exactly **one team per email domain**.

On signup, users verify their email address. If a team already exists for their domain, they are offered to join it automatically. If no team exists, they can create one or join via an invite code.

Admins can invite members using a **one-time, 8-character invite code**, generated for a specific email address. Each code can be redeemed only once. Contractors or non-domain users (e.g. Gmail) can join teams via this mechanism.

Roles are intentionally minimal:

- **Admin**: can invite members, remove members, and manage roles
- **Member**: can only manage their own sharing preferences

There is no “owner” role. Admins can promote or demote others, including themselves.

Teams are capped at **30 members** in v1 to keep the menu-bar experience usable.

## Timezone sharing model

Timezone sharing is opt-in.

When someone joins a team, they are asked whether they want to share their timezone. If they decline, they simply do not appear in the menu-bar list. There is no placeholder or “unknown” state shown to others.

Once enabled, timezone updates happen automatically. Users never manually set or update their timezone. The most recently reported timezone is always treated as the source of truth.

## Working hours and UI hints

During onboarding, users can optionally configure their typical working hours (for example, 11 AM–6 PM), with the same hours applying every day. Saturday and Sunday can be disabled individually.

These hours are interpreted strictly in the user’s **own local timezone**. There is no conversion or adjustment logic beyond that.

The app uses this information only for lightweight UI hints (such as showing a moon icon when someone is outside their working hours). There is no availability status, scheduling logic, or notifications tied to working hours.

## Hiding and privacy controls

Users can temporarily hide their timezone sharing for:

- 7 days
- 15 days
- 30 days
- Or indefinitely, until they manually re-enable it

While hidden, they still appear to admins in a greyed-out state, showing their last known time and a “hidden until” indicator. Hidden users are always pushed to the bottom of the list.

Admins cannot override or disable someone’s hide setting. Privacy is strictly user-controlled.

If a user disables sharing indefinitely, they receive a gentle reminder notification every 30 days.

## Freshness and reliability

Timezone reporting is passive and continuous. The app is required to launch at login to ensure reliability. If this is later disabled, the UI shows a small badge indicating reduced reliability, but no intrusive notifications are sent.

There is no concept of “stale” or “offline” shown to other users. This is intentionally a calm, low-signal product.

## Data and privacy principles

TimeSync is privacy-first by design.

The product does **not**:

- Track location
- Store IP addresses
- Infer or guess timezone
- Store movement or travel history

The only data shared between teammates is the current timezone (derived country metadata included) and optional working hours. Country is derived automatically from timezone or transient IP data in memory, but IP addresses are never stored.

## What this product is not

TimeSync is not:

- A scheduling tool
- A calendar replacement
- A presence or availability tracker
- A productivity or monitoring tool

It is a passive awareness layer, designed to reduce friction without creating pressure or noise.

## Product philosophy

TimeSync is intentionally narrow.

By solving one problem extremely well — knowing what time it is for someone — it earns a permanent place in the menu bar. Everything that does not directly support that goal is out of scope.
