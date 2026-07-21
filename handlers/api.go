package handlers

import (
	"database/sql"
	"encoding/json"
	"logistictbot/db"
	"logistictbot/errlog"
	"logistictbot/parser"
	"net/http"
	"strconv"
)

func RequestShipment(w http.ResponseWriter, r *http.Request, u *db.User, globalStorage *sql.DB) {
	idStr := r.PathValue("id")
	shipmentId, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid shipment id", http.StatusBadRequest)
		return
	}

	shipment, err := parser.GetShipment(globalStorage, shipmentId)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "shipment not found", http.StatusNotFound)
			return
		}
		errlog.ERR.Printf("get shipment %d: %v\n", shipmentId, err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	if !CanAccessShipment(u, shipment, globalStorage) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(shipment)
}

func RequestUpdateShipment(w http.ResponseWriter, r *http.Request, u *db.User, globalStorage *sql.DB) {
	idStr := r.PathValue("id")
	shipmentId, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid shipment id", http.StatusBadRequest)
		return
	}

	// Load current state first — needed for the authorization check,
	// and so we know what actually belongs to this shipment before
	// trusting anything from the client.
	existing, err := parser.GetShipment(globalStorage, shipmentId)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "shipment not found", http.StatusNotFound)
			return
		}
		errlog.ERR.Printf("get shipment %d: %v\n", shipmentId, err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if !CanAccessShipment(u, existing, globalStorage) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	var payload parser.UpdateShipmentInput
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&payload); err != nil {
		http.Error(w, "invalid body: "+err.Error(), http.StatusBadRequest)
		return
	}

	if err := payload.Validate(); err != nil {
		http.Error(w, "validation failed: "+err.Error(), http.StatusBadRequest)
		return
	}

	updated, err := parser.UpdateShipment(globalStorage, shipmentId, payload)
	if err != nil {
		errlog.ERR.Printf("update shipment %d: %v\n", shipmentId, err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(updated)
}

func CanAccessShipment(u *db.User, s *parser.Shipment, globalStorage *sql.DB) bool {
	ok, err := u.IsManager(globalStorage)
	if err != nil {
		errlog.ERR.Printf("%s (%s) not a manager, cannot update shipment (%d)\n", u.Name, u.Id.String(), s.Id)
		return false
	}
	return ok
}
