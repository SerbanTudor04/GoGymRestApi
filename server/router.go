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
}

func (app *App) setupUserRouter(r *mux.Router) {
	user := r.PathPrefix("/users").Subrouter()
	// Protected route - requires JWT authentication
	user.HandleFunc("/me", app.authenticateJWT(app.getMe)).Methods("GET")
	user.HandleFunc("/register", app.registerUser).Methods("POST")
	user.HandleFunc("/login", app.loginUser).Methods("POST")
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

func (app *App) setupGymsRouter(r *mux.Router) {
	g := r.PathPrefix("/gyms").Subrouter()
	g.Use(app.authenticateJWTMiddleware)
	g.HandleFunc("/create", app.createGym).Methods("POST")
	g.HandleFunc("/", app.getGyms).Methods("GET")
	g.HandleFunc("/add-user", app.addUserToGym).Methods("POST")
	g.HandleFunc("/{gym_id}/users/{user_id}", app.addUserToGymByPath).Methods("POST")
	// For JSON body version
	g.HandleFunc("/membership/add", app.addMembershipToGym).Methods("POST")

	// For path parameter version
	g.HandleFunc("/{gym_id}/membership/{membership_id}", app.addMembershipToGymByPath).Methods("POST")

	// For JSON body version
	g.HandleFunc("/machine/add", app.addMachineToGym).Methods("POST")

	// For path parameter version
	g.HandleFunc("/{gym_id}/machine/{machine_id}", app.addMachineToGymByPath).Methods("POST")
	// For gym stats retrieval
	g.HandleFunc("/api/gym/{gym_id}/stats", app.getGymStats).Methods("GET")

}

func (app *App) setupClientsRouter(r *mux.Router) {
	c := r.PathPrefix("/clients").Subrouter()
	c.Use(app.authenticateJWTMiddleware)
	c.HandleFunc("/", app.getClients).Methods("GET")
	c.HandleFunc("/add-user", app.addUserToClient).Methods("POST")
	c.HandleFunc("/{client_id}/users/{user_id}", app.addUserToClientByPath).Methods("POST")
	c.HandleFunc("/create", app.createClient).Methods("POST")
	// For JSON body version
	c.HandleFunc("/membership/add", app.addClientMembership).Methods("POST")

	// For path parameter version
	c.HandleFunc("/{client_id}/membership/{membership_id}/from/{valid_from}", app.addClientMembershipByPath).Methods("POST")
	// For JSON body version
	c.HandleFunc("/checkin", app.doClientCheckInGym).Methods("POST")
	// For path parameter version
	c.HandleFunc("/{client_id}/checkin/gym/{gym_id}", app.doClientCheckInGymByPath).Methods("POST")

}
