package server

import "github.com/gorilla/mux"

func (app *App) setupApiRouter(r *mux.Router) {

	// Routes
	api := r.PathPrefix("/api").Subrouter()
	app.setupUserRouter(api)
	app.setupNomenclatorsRouter(api)
	app.setupMembershipsRouter(api)
	app.setupGymsRouter(api)
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

}
