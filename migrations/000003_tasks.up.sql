CREATE TABLE tasks (
    id          BIGINT UNSIGNED                                          NOT NULL AUTO_INCREMENT,
    team_id     BIGINT UNSIGNED                                          NOT NULL,
    title       VARCHAR(255)                                             NOT NULL,
    description TEXT                                                     NOT NULL,
    status      ENUM ('todo', 'in_progress', 'done', 'cancelled')        NOT NULL DEFAULT 'todo',
    created_by  BIGINT UNSIGNED                                          NOT NULL,
    assignee_id BIGINT UNSIGNED                                          NULL,
    created_at  DATETIME(6)                                              NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    updated_at  DATETIME(6)                                              NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    closed_at   DATETIME(6)                                              NULL,
    version     INT UNSIGNED                                             NOT NULL DEFAULT 1,
    PRIMARY KEY (id),
    KEY idx_tasks_team_id (team_id, id),
    KEY idx_tasks_team_status (team_id, status, id),
    KEY idx_tasks_team_assignee (team_id, assignee_id, id),
    KEY idx_tasks_team_closed (team_id, closed_at),
    CONSTRAINT fk_tasks_team FOREIGN KEY (team_id) REFERENCES teams (id) ON DELETE CASCADE,
    CONSTRAINT fk_tasks_created_by FOREIGN KEY (created_by) REFERENCES users (id),
    CONSTRAINT fk_tasks_assignee FOREIGN KEY (assignee_id) REFERENCES users (id)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_0900_ai_ci;

CREATE TABLE task_history (
    id         BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    task_id    BIGINT UNSIGNED NOT NULL,
    changed_by BIGINT UNSIGNED NOT NULL,
    changes    JSON            NOT NULL,
    created_at DATETIME(6)     NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    PRIMARY KEY (id),
    KEY idx_task_history_task (task_id, id),
    CONSTRAINT fk_task_history_task FOREIGN KEY (task_id) REFERENCES tasks (id) ON DELETE CASCADE,
    CONSTRAINT fk_task_history_user FOREIGN KEY (changed_by) REFERENCES users (id)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_0900_ai_ci;