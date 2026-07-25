INSERT INTO users (
  email, password_hash, first_name, last_name, date_of_birth,
  gender, nickname, about_me, avatar_media_id, is_private,
  created_at, updated_at
)
SELECT
  'alice.demo@example.com', '{{DEMO_PASSWORD_HASH}}', 'Alice', 'Demo', '02-02-1986',
  'female', 'alice-demo', 'Public demo profile', NULL, 0,
  unixepoch() - 300, unixepoch() - 300
WHERE NOT EXISTS (
  SELECT 1 FROM users WHERE email = 'alice.demo@example.com' COLLATE NOCASE
);

INSERT INTO users (
  email, password_hash, first_name, last_name, date_of_birth,
  gender, nickname, about_me, avatar_media_id, is_private,
  created_at, updated_at
)
SELECT
  'bob.demo@example.com', '{{DEMO_PASSWORD_HASH}}', 'Bob', 'Demo', '14-03-1990',
  'male', 'bob-demo', 'Private demo profile', NULL, 1,
  unixepoch() - 200, unixepoch() - 200
WHERE NOT EXISTS (
  SELECT 1 FROM users WHERE email = 'bob.demo@example.com' COLLATE NOCASE
);

INSERT INTO users (
  email, password_hash, first_name, last_name, date_of_birth,
  gender, nickname, about_me, avatar_media_id, is_private,
  created_at, updated_at
)
SELECT
  'carol.demo@example.com', '{{DEMO_PASSWORD_HASH}}', 'Carol', 'Demo', '24-07-1992',
  'female', 'carol-demo', 'Public demo profile', NULL, 0,
  unixepoch() - 100, unixepoch() - 100
WHERE NOT EXISTS (
  SELECT 1 FROM users WHERE email = 'carol.demo@example.com' COLLATE NOCASE
);

INSERT INTO notification_user_states (user_id, revision)
SELECT id, 0
FROM users
WHERE email LIKE '%.demo@example.com'
ON CONFLICT(user_id) DO NOTHING;

INSERT INTO chat_user_states (user_id, revision)
SELECT id, 0
FROM users
WHERE email LIKE '%.demo@example.com'
ON CONFLICT(user_id) DO NOTHING;

INSERT INTO follows (
  follower_user_id, followed_user_id, status, created_at, updated_at
)
SELECT bob.id, alice.id, 'accepted', unixepoch() - 90, unixepoch() - 90
FROM users bob
JOIN users alice
  ON alice.email = 'alice.demo@example.com' COLLATE NOCASE
WHERE bob.email = 'bob.demo@example.com' COLLATE NOCASE
  AND NOT EXISTS (
    SELECT 1
    FROM follows existing
    WHERE existing.follower_user_id = bob.id
      AND existing.followed_user_id = alice.id
  );

INSERT INTO follows (
  follower_user_id, followed_user_id, status, created_at, updated_at
)
SELECT carol.id, alice.id, 'accepted', unixepoch() - 80, unixepoch() - 80
FROM users carol
JOIN users alice
  ON alice.email = 'alice.demo@example.com' COLLATE NOCASE
WHERE carol.email = 'carol.demo@example.com' COLLATE NOCASE
  AND NOT EXISTS (
    SELECT 1
    FROM follows existing
    WHERE existing.follower_user_id = carol.id
      AND existing.followed_user_id = alice.id
  );

INSERT INTO posts (
  author_user_id, group_id, text, privacy, media_id, created_at
)
SELECT id, NULL, 'A public demo post', 'public', NULL, unixepoch() - 70
FROM users
WHERE email = 'alice.demo@example.com' COLLATE NOCASE
  AND NOT EXISTS (
    SELECT 1
    FROM posts existing
    WHERE existing.author_user_id = users.id
      AND existing.group_id IS NULL
      AND existing.text = 'A public demo post'
      AND existing.privacy = 'public'
  );

INSERT INTO posts (
  author_user_id, group_id, text, privacy, media_id, created_at
)
SELECT id, NULL, 'A followers-only demo post', 'followers', NULL, unixepoch() - 60
FROM users
WHERE email = 'alice.demo@example.com' COLLATE NOCASE
  AND NOT EXISTS (
    SELECT 1
    FROM posts existing
    WHERE existing.author_user_id = users.id
      AND existing.group_id IS NULL
      AND existing.text = 'A followers-only demo post'
      AND existing.privacy = 'followers'
  );

INSERT INTO posts (
  author_user_id, group_id, text, privacy, media_id, created_at
)
SELECT id, NULL, 'A selected-audience demo post', 'selected', NULL, unixepoch() - 50
FROM users
WHERE email = 'alice.demo@example.com' COLLATE NOCASE
  AND NOT EXISTS (
    SELECT 1
    FROM posts existing
    WHERE existing.author_user_id = users.id
      AND existing.group_id IS NULL
      AND existing.text = 'A selected-audience demo post'
      AND existing.privacy = 'selected'
  );

INSERT INTO post_selected_users (post_id, user_id)
SELECT post.id, bob.id
FROM posts post
JOIN users alice ON alice.id = post.author_user_id
JOIN users bob ON bob.email = 'bob.demo@example.com' COLLATE NOCASE
WHERE alice.email = 'alice.demo@example.com' COLLATE NOCASE
  AND post.group_id IS NULL
  AND post.text = 'A selected-audience demo post'
  AND post.privacy = 'selected'
  AND NOT EXISTS (
    SELECT 1
    FROM post_selected_users existing
    WHERE existing.post_id = post.id
      AND existing.user_id = bob.id
  );

INSERT INTO groups (owner_user_id, title, description, created_at)
SELECT id, 'Loop Demo Group', 'Group data created by the opt-in demo seed.', unixepoch() - 40
FROM users
WHERE email = 'alice.demo@example.com' COLLATE NOCASE
  AND NOT EXISTS (
    SELECT 1
    FROM groups existing
    WHERE existing.owner_user_id = users.id
      AND existing.title = 'Loop Demo Group'
  );

INSERT INTO group_memberships (
  group_id, user_id, status, created_at, updated_at
)
SELECT demo_group.id, demo_user.id, membership.status, unixepoch() - 35, unixepoch() - 35
FROM groups demo_group
JOIN users owner
  ON owner.id = demo_group.owner_user_id
JOIN (
  SELECT 'alice.demo@example.com' AS email, 'owner' AS status
  UNION ALL
  SELECT 'bob.demo@example.com', 'member'
  UNION ALL
  SELECT 'carol.demo@example.com', 'member'
) membership
JOIN users demo_user
  ON demo_user.email = membership.email COLLATE NOCASE
WHERE owner.email = 'alice.demo@example.com' COLLATE NOCASE
  AND demo_group.title = 'Loop Demo Group'
  AND NOT EXISTS (
    SELECT 1
    FROM group_memberships existing
    WHERE existing.group_id = demo_group.id
      AND existing.user_id = demo_user.id
  );

INSERT INTO group_chat_read_states (
  membership_id, last_read_message_id, unread_count, updated_at
)
SELECT membership.id, NULL, 0, membership.updated_at
FROM group_memberships membership
JOIN groups demo_group ON demo_group.id = membership.group_id
JOIN users owner ON owner.id = demo_group.owner_user_id
WHERE owner.email = 'alice.demo@example.com' COLLATE NOCASE
  AND demo_group.title = 'Loop Demo Group'
  AND membership.status IN ('owner', 'member')
ON CONFLICT(membership_id) DO NOTHING;

INSERT INTO posts (
  author_user_id, group_id, text, privacy, media_id, created_at
)
SELECT bob.id, demo_group.id, 'A demo group post', NULL, NULL, unixepoch() - 30
FROM users bob
JOIN groups demo_group
JOIN users owner ON owner.id = demo_group.owner_user_id
WHERE bob.email = 'bob.demo@example.com' COLLATE NOCASE
  AND owner.email = 'alice.demo@example.com' COLLATE NOCASE
  AND demo_group.title = 'Loop Demo Group'
  AND NOT EXISTS (
    SELECT 1
    FROM posts existing
    WHERE existing.author_user_id = bob.id
      AND existing.group_id = demo_group.id
      AND existing.text = 'A demo group post'
  );

INSERT INTO group_events (
  group_id, creator_user_id, title, description, starts_at, created_at
)
SELECT
  demo_group.id, owner.id, 'Demo community call',
  'A future event created by the opt-in demo seed.',
  unixepoch() + 604800, unixepoch() - 20
FROM groups demo_group
JOIN users owner ON owner.id = demo_group.owner_user_id
WHERE owner.email = 'alice.demo@example.com' COLLATE NOCASE
  AND demo_group.title = 'Loop Demo Group'
  AND NOT EXISTS (
    SELECT 1
    FROM group_events existing
    WHERE existing.group_id = demo_group.id
      AND existing.title = 'Demo community call'
  );

INSERT INTO group_event_responses (
  event_id, user_id, response, created_at, updated_at
)
SELECT event.id, bob.id, 'going', unixepoch() - 10, unixepoch() - 10
FROM group_events event
JOIN groups demo_group ON demo_group.id = event.group_id
JOIN users owner ON owner.id = demo_group.owner_user_id
JOIN users bob ON bob.email = 'bob.demo@example.com' COLLATE NOCASE
WHERE owner.email = 'alice.demo@example.com' COLLATE NOCASE
  AND demo_group.title = 'Loop Demo Group'
  AND event.title = 'Demo community call'
  AND NOT EXISTS (
    SELECT 1
    FROM group_event_responses existing
    WHERE existing.event_id = event.id
      AND existing.user_id = bob.id
  );
