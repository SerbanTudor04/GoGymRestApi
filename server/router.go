package server

import "github.com/gorilla/mux"

func (app *App) setupApiRouter(r *mux.Router) {

	// Routes
	api := r.PathPrefix("/api").Subrouter()
	app.setupUserRouter(api)
	app.setupNomenclatorsRouter(api)
	app.setupMembershipsRouter(api)
	app.setupGymsRouter(api)
	app.setupClientsRouter(api)
	api.HandleFunc("/health", app.healthCheck).Methods("GET")
}

// Update your setupUserRouter function in router.go

func (app *App) setupUserRouter(r *mux.Router) {
	user := r.PathPrefix("/users").Subrouter()

	// Public routes (no authentication required)
	user.HandleFunc("/register", app.registerUser).Methods("POST")
	user.HandleFunc("/login", app.loginUser).Methods("POST")

	// Protected routes (require JWT authentication)
	user.HandleFunc("/me", app.authenticateJWT(app.getMe)).Methods("GET")
	user.HandleFunc("/", app.authenticateJWT(app.getUsers)).Methods("GET")
	user.HandleFunc("/search", app.authenticateJWT(app.getUsersWithSearch)).Methods("GET")
}

func (app *App) setupNomenclatorsRouter(r *mux.Router) {
	nom := r.PathPrefix("/nomenclators").Subrouter()
	nom.Use(app.authenticateJWTMiddleware)
	nom.HandleFunc("/countries", app.getCountries).Methods("GET")
	nom.HandleFunc("/states", app.getStates).Methods("GET")
}

func (app *App) setupMembershipsRouter(r *mux.Router) {
	m := r.PathPrefix("/memberships").Subrouter()
	m.Use(app.authenticateJWTMiddleware)
	m.HandleFunc("/", app.getMemberships).Methods("GET")
}

// Add these routes to your setupGymsRouter function in router.go

func (app *App) setupGymsRouter(r *mux.Router) {
	g := r.PathPrefix("/gyms").Subrouter()
	g.Use(app.authenticateJWTMiddleware)

	// Basic CRUD operations
	g.HandleFunc("/create", app.createGym).Methods("POST")
	g.HandleFunc("/", app.getGyms).Methods("GET")
	g.HandleFunc("/{gym_id}", app.updateGym).Methods("PUT")
	g.HandleFunc("/{gym_id}", app.deleteGym).Methods("DELETE")

	// User management
	g.HandleFunc("/add-user", app.addUserToGym).Methods("POST")
	g.HandleFunc("/{gym_id}/users/{user_id}", app.addUserToGymByPath).Methods("POST")
	g.HandleFunc("/{gym_id}/users/{user_id}", app.removeUserFromGym).Methods("DELETE")

	// Membership management
	g.HandleFunc("/membership/add", app.addMembershipToGym).Methods("POST")
	g.HandleFunc("/{gym_id}/membership/{membership_id}", app.addMembershipToGymByPath).Methods("POST")
	g.HandleFunc("/{gym_id}/membership/{membership_id}", app.removeMembershipFromGym).Methods("DELETE")

	// Machine management
	g.HandleFunc("/machine/add", app.addMachineToGym).Methods("POST")
	g.HandleFunc("/{gym_id}/machine/{machine_id}", app.addMachineToGymByPath).Methods("POST")
	g.HandleFunc("/{gym_id}/machine/{machine_id}", app.removeMachineFromGym).Methods("DELETE")

	// Stats
	g.HandleFunc("/{gym_id}/stats", app.getGymStats).Methods("GET")
}

// Add these routes to your setupClientsRouter function in router.go

func (app *App) setupClientsRouter(r *mux.Router) {
	c := r.PathPrefix("/clients").Subrouter()
	c.Use(app.authenticateJWTMiddleware)

	// Basic CRUD operations
	c.HandleFunc("/", app.getClients).Methods("GET")
	c.HandleFunc("/create", app.createClient).Methods("POST")
	c.HandleFunc("/{client_id}", app.getClientByID).Methods("GET")
	c.HandleFunc("/{client_id}", app.updateClient).Methods("PUT")
	c.HandleFunc("/{client_id}", app.deleteClient).Methods("DELETE")

	// User management
	c.HandleFunc("/add-user", app.addUserToClient).Methods("POST")
	c.HandleFunc("/{client_id}/users/{user_id}", app.addUserToClientByPath).Methods("POST")
	c.HandleFunc("/{client_id}/users/{user_id}", app.removeUserFromClient).Methods("DELETE")

	// Membership management
	c.HandleFunc("/membership/add", app.addClientMembership).Methods("POST")
	c.HandleFunc("/{client_id}/membership/{membership_id}/from/{valid_from}", app.addClientMembershipByPath).Methods("POST")
	c.HandleFunc("/{client_id}/membership/{membership_id}", app.removeClientMembership).Methods("DELETE")
	c.HandleFunc("/{client_id}/membership/{membership_id}/deactivate", app.deactivateClientMembership).Methods("PATCH")

	// Check-in/Check-out
	c.HandleFunc("/checkin", app.doClientCheckInGym).Methods("POST")
	c.HandleFunc("/{client_id}/checkin/gym/{gym_id}", app.doClientCheckInGymByPath).Methods("POST")
	c.HandleFunc("/checkout", app.doClientCheckOutGym).Methods("POST")
	c.HandleFunc("/{client_id}/checkout/gym/{gym_id}", app.doClientCheckOutGymByPath).Methods("POST")

	// Status check
	c.HandleFunc("/{client_id}/gym/{gym_id}/status", app.getClientGymStatus).Methods("GET")
}
