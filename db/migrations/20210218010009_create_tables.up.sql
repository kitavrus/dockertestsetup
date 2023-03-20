CREATE TABLE promocodes (
  id          bigserial  PRIMARY KEY,
  description VARCHAR NOT NULL,
  start_date  TIMESTAMP WITH TIME ZONE,
  due_date    TIMESTAMP WITH TIME ZONE,
  status      integer DEFAULT 0
);