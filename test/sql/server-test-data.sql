insert into user_record (inner_id, id, name, session_token, refresh_token, token_expire, id_token, profile, environment, organization_id, organization_name, updated_at)
values (1, 'N:user:8888', 'Harry Proctor', 'session-token', 'refresh-token', '2024-09-27 14:37:10-04:00', 'id-token', 'profile-1', '', 'N:organization:1111', 'Organization 1', '2022-11-22 16:23:04.895151+00:00');

INSERT INTO user_settings (user_id, profile, use_dataset_id)
VALUES ('N:user:8888','profile-1','N:dataset:1');
