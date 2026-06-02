-- Migration 004 — Adiciona coluna name na tabela users
ALTER TABLE lume_system.users ADD COLUMN IF NOT EXISTS name VARCHAR(255) NOT NULL DEFAULT '';