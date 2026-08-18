-- Remove seeded green_tags
DELETE FROM green_tags WHERE postgres_major = 16 AND postgres_minor = 4;
