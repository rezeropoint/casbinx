package role

import (
	"database/sql"
	"fmt"

	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

// roleMetadata 角色元数据结构体
type roleMetadata struct {
	RoleKey     string         `db:"role_key"`
	Name        string         `db:"name"`
	Description sql.NullString `db:"description"`
	TenantKey   string         `db:"tenant_key"`
	CreatedAt   sql.NullTime   `db:"created_at"`
	UpdatedAt   sql.NullTime   `db:"updated_at"`
	CreatedBy   sql.NullString `db:"created_by"`
}

// initDB 初始化数据库，创建角色元数据表
func initDB(dbConn sqlx.SqlConn) error {
	// 先检查表是否存在，如果存在就跳过创建
	exists, err := tableExists(dbConn, "system_roles")
	if err != nil {
		return fmt.Errorf("检查system_roles表是否存在失败: %v", err)
	}

	if exists {
		return nil // 表已存在，无需创建
	}

	// 表不存在，创建表和索引
	createRolesTableSQL := `
CREATE TABLE system_roles (
    role_key VARCHAR(255) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    tenant_key VARCHAR(255) NOT NULL DEFAULT '*',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    created_by VARCHAR(255),
    UNIQUE(role_key, tenant_key)
);

CREATE INDEX idx_system_roles_tenant_key ON system_roles(tenant_key);
CREATE INDEX idx_system_roles_created_at ON system_roles(created_at);
`

	_, err = dbConn.Exec(createRolesTableSQL)
	if err != nil {
		return fmt.Errorf("创建system_roles表失败: %v", err)
	}

	return nil
}

// tableExists 检查表是否存在
func tableExists(dbConn sqlx.SqlConn, tableName string) (bool, error) {
	var exists bool
	checkSQL := `
		SELECT EXISTS (
			SELECT 1 FROM information_schema.tables
			WHERE table_schema = 'public' AND table_name = $1
		)
	`
	err := dbConn.QueryRow(&exists, checkSQL, tableName)
	return exists, err
}
