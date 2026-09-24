package model

import "time"

const SessionTokenDigestSize = 32

type Session struct {
	ID          uint      `gorm:"primaryKey"`
	UserID      uint      `gorm:"not null;index:idx_sessions_user_id"`
	User        User      `gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT"`
	TokenDigest []byte    `gorm:"type:bytea;not null;uniqueIndex:ux_sessions_token_digest;check:chk_sessions_token_digest_length,octet_length(token_digest) = 32"`
	ExpiresAt   time.Time `gorm:"not null;index:idx_sessions_expires_at"`
	CreatedAt   time.Time `gorm:"not null;autoCreateTime"`
}
