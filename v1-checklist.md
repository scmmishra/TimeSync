This is the scope required to ship a clean, opinionated v1 — nothing more.

### Core experience

- macOS menu-bar app only (no Dock icon)
- Show teammates’ exact local time
- Sort list by time difference from the viewer
- Friendly display: avatar with flag, name, local time, country + short zone label
- Subtle “outside working hours” visual hint

### Teams & onboarding

- Email verification via one-time 8-character code
- One team per email domain
- Domain users can auto-join existing team after verification
- Admins can generate one-time invite codes bound to a specific email
- Invite codes are single-use
- Team size capped at 30 members

### Roles

- Two roles only: admin and member
- Admins can invite, remove members, and change roles
- No owner role

### Timezone sharing

- Sharing is opt-in
- Users who don’t share do not appear in the list
- Timezone updates happen automatically
- Most recently reported timezone is always authoritative

### Working hours

- Optional setup during onboarding
- One daily window (e.g., 11 AM–6 PM)
- Same hours every day
- Saturday and Sunday can be disabled
- Used only for UI hints

### Hide / privacy controls

- Hide timezone for 7, 15, or 30 days
- Hide indefinitely until manually re-enabled
- Hidden users appear greyed out to admins, pushed to bottom
- Admins cannot override hide
- Reminder notification every 30 days if disabled indefinitely

### Reliability

- Launch at login required
- UI badge if launch-at-login is turned off
- No “stale” or “offline” markers shown to teammates

### Explicitly out of scope

- Mobile apps
- Scheduling or meeting tools
- Search or filtering
- Multiple teams per domain
- Analytics or activity tracking
