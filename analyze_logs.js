const statusStats = db.traffic_logs.aggregate([
    { $match: { status: { $gte: 400 } } },
    { $group: { _id: '$status', count: { $sum: 1 }, sample_msg: { $first: '$message' } } }
]).toArray();
printjson(statusStats);
