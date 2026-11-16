INSERT INTO platforms (issuer, client_id, deployment_id, auth_login_url, jwks_url)
VALUES ('https://dev.example', 'dev-client', 'dev-deployment',
        'https://dev.example/auth', 'https://dev.example/jwks');

INSERT INTO courses (platform_id, context_id, title)
VALUES (1, 'dev-course', 'Dev Course');