package authz

# Allow access if the user is assigned to the room
allow if {
    # Query the database to check if user has access to the room
    results := postgres.query("SELECT * FROM room_access WHERE room_id = $1 AND user_id = $2", [input.room_id, input.user_id])
    count(results) > 0
}
