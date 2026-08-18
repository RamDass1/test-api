CREATE TABLE teams (
    id         BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    name       VARCHAR(255)    NOT NULL,
    created_by BIGINT UNSIGNED NOT NULL,
    created_at DATETIME(6)     NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    PRIMARY KEY (id),
    KEY idx_teams_created_by (created_by),
    CONSTRAINT fk_teams_created_by FOREIGN KEY (created_by) REFERENCES users (id)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_0900_ai_ci;
  
CREATE TABLE team_members (
    team_id    BIGINT UNSIGNED                   NOT NULL,
    user_id    BIGINT UNSIGNED                   NOT NULL,
    role       ENUM ('owner', 'admin', 'member') NOT NULL,
    created_at DATETIME(6)                       NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    PRIMARY KEY (team_id, user_id),
    KEY idx_team_members_user (user_id),
    CONSTRAINT fk_team_members_team FOREIGN KEY (team_id) REFERENCES teams (id) ON DELETE CASCADE,
    CONSTRAINT fk_team_members_user FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_0900_ai_ci;