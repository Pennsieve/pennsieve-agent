INSERT INTO ts_channel (node_id, package_id, name, start_time, end_time, unit, rate) VALUES
('N:channel:1', 'N:package:1', 'A1', 1, 500, 'uV', 2048),
('N:channel:2', 'N:package:1', 'B1', 1, 500, 'uV', 2048),
('N:channel:3', 'N:package:1', 'C1', 1, 500, 'uV', 2048),
('N:channel:4', 'N:package:2', 'A1', 1, 500, 'uV', 2048),
('N:channel:5', 'N:package:2', 'B1', 1, 500, 'uV', 2048);

INSERT INTO ts_range (node_id, channel_node_id, location, start_time, end_time) VALUES
('19-1','N:channel:1', 'location/1', 1, 100),
('19-2','N:channel:1', 'location/2', 100, 150),
('19-3','N:channel:2', 'location/3', 100, 200),
('19-4','N:channel:3', 'location/4', 1, 100),
('19-5','N:channel:4', 'location/5', 1, 100),
('19-6','N:channel:5', 'location/6', 1, 100);
