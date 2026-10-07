package admin

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"

	"ecommerce-cli/internal/api"
	"ecommerce-cli/internal/apiclient"
	"ecommerce-cli/ui/style"
)

func Run(serverAddr string) error {
	m := newModel(serverAddr)
	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err := p.Run()
	return err
}

type screen int

const (
	screenAuth screen = iota
	screenMenu
	screenProducts
	screenAddProduct
	screenOrders
	screenOrderStatus
	screenCreateOrder
	screenUsers
	screenAddUser
	screenEditUser
)

type simpleItem struct {
	title, desc string
}

func (i simpleItem) Title() string       { return i.title }
func (i simpleItem) Description() string { return i.desc }
func (i simpleItem) FilterValue() string { return i.title }

type productItem struct{ p api.ProductDTO }

func (i productItem) Title() string {
	return fmt.Sprintf("#%d [%s] %s", i.p.ID, i.p.Reference, i.p.Name)
}
func (i productItem) Description() string {
	return fmt.Sprintf("%.2f€ HT / %.2f€ TTC · stock: %d · %s", float64(i.p.PriceCents)/100, float64(i.p.PriceCentsTTC)/100, i.p.Stock, i.p.Category)
}
func (i productItem) FilterValue() string { return i.p.Name + " " + i.p.Category }

type orderItem struct{ o api.OrderDTO }

func (i orderItem) Title() string {
	return fmt.Sprintf("Commande #%d - %s", i.o.ID, i.o.Status)
}
func (i orderItem) Description() string {
	return fmt.Sprintf("utilisateur #%d · %.2f€ · %d article(s)", i.o.UserID, float64(i.o.TotalCents)/100, len(i.o.Items))
}
func (i orderItem) FilterValue() string { return fmt.Sprintf("%d", i.o.ID) }

type userItem struct{ u api.UserDTO }

func (i userItem) Title() string {
	return fmt.Sprintf("#%d %s", i.u.ID, i.u.Email)
}
func (i userItem) Description() string {
	return fmt.Sprintf("admin: %t · confirmé: %t", i.u.IsAdmin, i.u.Confirmed)
}
func (i userItem) FilterValue() string { return i.u.Email }

type errMsg struct{ err error }
type authDoneMsg struct{ resp *api.AuthResponse }
type productsLoadedMsg struct{ products []api.ProductDTO }
type ordersLoadedMsg struct{ orders []api.OrderDTO }
type usersLoadedMsg struct{ users []api.UserDTO }
type actionOKMsg struct{ message string }

type model struct {
	api    *apiclient.Client
	screen screen
	width  int
	height int

	user *api.UserDTO

	authForm *huh.Form
	authMode string
	email    string
	password string

	menuList list.Model
	products list.Model
	orders   list.Model
	users    list.Model

	addProductForm *huh.Form
	newName        string
	newDesc        string
	newCategory    string
	newPrice       string
	newStock       string

	orderStatusForm *huh.Form
	targetOrderID   int64
	orderStatus     string

	createOrderForm *huh.Form
	orderEmail      string
	orderProductID  string
	orderQuantity   string

	addUserForm  *huh.Form
	newUserEmail string
	newUserPass  string
	newUserAdmin bool

	editUserForm      *huh.Form
	targetUserID      int64
	editEmail         string
	editPassword      string
	editAdminChoice   string
	editOriginalAdmin bool

	statusMsg string
	statusErr bool
	loading   bool
}

func newModel(serverAddr string) *model {
	m := &model{
		api:      apiclient.New(serverAddr),
		screen:   screenAuth,
		authMode: "login",
	}
	m.authForm = m.buildAuthForm()
	m.menuList = newMenuList()
	m.products = newList("Produits")
	m.orders = newList("Commandes")
	m.users = newList("Utilisateurs")
	return m
}

func newMenuList() list.Model {
	items := []list.Item{
		simpleItem{"Gérer les produits", "Lister, ajouter, supprimer"},
		simpleItem{"Gérer les commandes", "Lister, changer le statut, créer pour un client"},
		simpleItem{"Gérer les utilisateurs", "Lister, créer, modifier, confirmer, supprimer"},
		simpleItem{"Quitter", "Fermer l'application"},
	}
	l := list.New(items, list.NewDefaultDelegate(), 0, 0)
	l.Title = "Menu admin"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.Styles.Title = style.Title
	return l
}

func newList(title string) list.Model {
	l := list.New(nil, list.NewDefaultDelegate(), 0, 0)
	l.Title = title
	l.Styles.Title = style.Title
	return l
}

func (m *model) buildAuthForm() *huh.Form {
	return huh.NewForm(
		huh.NewGroup(
			huh.NewNote().Title("Connexion administrateur"),
			huh.NewInput().
				Title("Email").
				Value(&m.email).
				Validate(func(s string) error {
					if s == "" {
						return fmt.Errorf("email requis")
					}
					return nil
				}),
			huh.NewInput().
				Title("Mot de passe").
				EchoMode(huh.EchoModePassword).
				Value(&m.password).
				Validate(func(s string) error {
					if len(s) < 4 {
						return fmt.Errorf("mot de passe trop court")
					}
					return nil
				}),
		),
	).WithWidth(60)
}

func (m *model) Init() tea.Cmd {
	return m.authForm.Init()
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		listHeight := msg.Height - 8
		if listHeight < 5 {
			listHeight = 5
		}
		m.menuList.SetSize(msg.Width-4, listHeight)
		m.products.SetSize(msg.Width-4, listHeight)
		m.orders.SetSize(msg.Width-4, listHeight)
		m.users.SetSize(msg.Width-4, listHeight)
		return m, nil

	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}

	case errMsg:
		m.loading = false
		m.statusErr = true
		m.statusMsg = msg.err.Error()
		return m, nil

	case actionOKMsg:
		m.loading = false
		m.statusErr = false
		m.statusMsg = msg.message
		return m, nil

	case authDoneMsg:
		m.loading = false
		if !msg.resp.User.IsAdmin {
			m.statusErr = true
			m.statusMsg = "Ce compte n'est pas administrateur."
			m.authForm = m.buildAuthForm()
			return m, m.authForm.Init()
		}
		m.user = &msg.resp.User
		m.api.SetToken(msg.resp.Token)
		m.statusErr = false
		m.statusMsg = fmt.Sprintf("Connecté en tant qu'admin (%s)", m.user.Email)
		m.screen = screenMenu
		return m, nil

	case productsLoadedMsg:
		m.loading = false
		items := make([]list.Item, 0, len(msg.products))
		for _, p := range msg.products {
			items = append(items, productItem{p})
		}
		m.products.SetItems(items)
		return m, nil

	case ordersLoadedMsg:
		m.loading = false
		items := make([]list.Item, 0, len(msg.orders))
		for _, o := range msg.orders {
			items = append(items, orderItem{o})
		}
		m.orders.SetItems(items)
		return m, nil

	case usersLoadedMsg:
		m.loading = false
		items := make([]list.Item, 0, len(msg.users))
		for _, u := range msg.users {
			items = append(items, userItem{u})
		}
		m.users.SetItems(items)
		return m, nil
	}

	switch m.screen {
	case screenAuth:
		return m.updateAuth(msg)
	case screenMenu:
		return m.updateMenu(msg)
	case screenProducts:
		return m.updateProducts(msg)
	case screenAddProduct:
		return m.updateAddProduct(msg)
	case screenOrders:
		return m.updateOrders(msg)
	case screenOrderStatus:
		return m.updateOrderStatus(msg)
	case screenCreateOrder:
		return m.updateCreateOrder(msg)
	case screenUsers:
		return m.updateUsers(msg)
	case screenAddUser:
		return m.updateAddUser(msg)
	case screenEditUser:
		return m.updateEditUser(msg)
	}
	return m, nil
}

func (m *model) updateAuth(msg tea.Msg) (tea.Model, tea.Cmd) {
	form, cmd := m.authForm.Update(msg)
	if f, ok := form.(*huh.Form); ok {
		m.authForm = f
	}
	if m.authForm.State == huh.StateCompleted {
		m.loading = true
		email, password := m.email, m.password
		m.authForm = m.buildAuthForm()
		return m, tea.Batch(cmd, m.doAuth(email, password))
	}
	return m, cmd
}

func (m *model) doAuth(email, password string) tea.Cmd {
	return func() tea.Msg {
		resp, err := m.api.Login(email, password)
		if err != nil {
			return errMsg{err}
		}
		return authDoneMsg{resp}
	}
}

func (m *model) updateMenu(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok && key.String() == "enter" {
		switch m.menuList.Index() {
		case 0:
			m.screen = screenProducts
			m.loading = true
			return m, m.loadProducts()
		case 1:
			m.screen = screenOrders
			m.loading = true
			return m, m.loadOrders()
		case 2:
			m.screen = screenUsers
			m.loading = true
			return m, m.loadUsers()
		case 3:
			return m, tea.Quit
		}
	}
	var cmd tea.Cmd
	m.menuList, cmd = m.menuList.Update(msg)
	return m, cmd
}

func (m *model) loadProducts() tea.Cmd {
	return func() tea.Msg {
		products, err := m.api.ListProducts()
		if err != nil {
			return errMsg{err}
		}
		return productsLoadedMsg{products}
	}
}

func (m *model) loadOrders() tea.Cmd {
	return func() tea.Msg {
		orders, err := m.api.ListAllOrders()
		if err != nil {
			return errMsg{err}
		}
		return ordersLoadedMsg{orders}
	}
}

func (m *model) loadUsers() tea.Cmd {
	return func() tea.Msg {
		users, err := m.api.ListUsers()
		if err != nil {
			return errMsg{err}
		}
		return usersLoadedMsg{users}
	}
}

func (m *model) updateProducts(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "esc":
			m.screen = screenMenu
			return m, nil
		case "a":
			m.addProductForm = m.buildAddProductForm()
			m.screen = screenAddProduct
			return m, m.addProductForm.Init()
		case "d":
			if it, ok := m.products.SelectedItem().(productItem); ok {
				m.loading = true
				return m, m.doDeleteProduct(it.p.ID)
			}
		}
	}
	var cmd tea.Cmd
	m.products, cmd = m.products.Update(msg)
	return m, cmd
}

func (m *model) buildAddProductForm() *huh.Form {
	m.newName, m.newDesc, m.newCategory, m.newPrice, m.newStock = "", "", "", "", ""
	return huh.NewForm(
		huh.NewGroup(
			huh.NewNote().Title("Nouveau produit"),
			huh.NewInput().Title("Nom").Value(&m.newName).
				Validate(nonEmpty("nom")),
			huh.NewInput().Title("Description").Value(&m.newDesc),
			huh.NewInput().Title("Catégorie").Value(&m.newCategory).
				Validate(nonEmpty("catégorie")),
			huh.NewInput().Title("Prix HT (en euros)").Value(&m.newPrice).
				Validate(validateFloat),
			huh.NewInput().Title("Stock").Value(&m.newStock).
				Validate(validateNonNegativeInt),
		),
	).WithWidth(60)
}

func nonEmpty(field string) func(string) error {
	return func(s string) error {
		if strings.TrimSpace(s) == "" {
			return fmt.Errorf("%s requis", field)
		}
		return nil
	}
}

func validateFloat(s string) error {
	if _, err := strconv.ParseFloat(s, 64); err != nil {
		return fmt.Errorf("prix invalide")
	}
	return nil
}

func validateNonNegativeInt(s string) error {
	n, err := strconv.Atoi(s)
	if err != nil || n < 0 {
		return fmt.Errorf("valeur invalide")
	}
	return nil
}

func validatePositiveInt(s string) error {
	n, err := strconv.Atoi(s)
	if err != nil || n <= 0 {
		return fmt.Errorf("valeur invalide")
	}
	return nil
}

func (m *model) updateAddProduct(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok && key.String() == "esc" {
		m.screen = screenProducts
		return m, nil
	}
	form, cmd := m.addProductForm.Update(msg)
	if f, ok := form.(*huh.Form); ok {
		m.addProductForm = f
	}
	if m.addProductForm.State == huh.StateCompleted {
		priceEuros, _ := strconv.ParseFloat(m.newPrice, 64)
		stock, _ := strconv.Atoi(m.newStock)
		req := api.CreateProductRequest{
			Name:        m.newName,
			Description: m.newDesc,
			Category:    m.newCategory,
			PriceCents:  int64(priceEuros * 100),
			Stock:       stock,
		}
		m.screen = screenProducts
		m.loading = true
		return m, tea.Batch(cmd, m.doCreateProduct(req))
	}
	return m, cmd
}

func (m *model) doCreateProduct(req api.CreateProductRequest) tea.Cmd {
	return func() tea.Msg {
		if _, err := m.api.CreateProduct(req); err != nil {
			return errMsg{err}
		}
		products, err := m.api.ListProducts()
		if err != nil {
			return errMsg{err}
		}
		return productsLoadedMsg{products}
	}
}

func (m *model) doDeleteProduct(id int64) tea.Cmd {
	return func() tea.Msg {
		if err := m.api.DeleteProduct(id); err != nil {
			return errMsg{err}
		}
		products, err := m.api.ListProducts()
		if err != nil {
			return errMsg{err}
		}
		return productsLoadedMsg{products}
	}
}

func (m *model) updateOrders(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "esc":
			m.screen = screenMenu
			return m, nil
		case "s":
			if it, ok := m.orders.SelectedItem().(orderItem); ok {
				m.targetOrderID = it.o.ID
				m.orderStatus = it.o.Status
				m.orderStatusForm = m.buildOrderStatusForm()
				m.screen = screenOrderStatus
				return m, m.orderStatusForm.Init()
			}
		case "n":
			m.createOrderForm = m.buildCreateOrderForm()
			m.screen = screenCreateOrder
			return m, m.createOrderForm.Init()
		}
	}
	var cmd tea.Cmd
	m.orders, cmd = m.orders.Update(msg)
	return m, cmd
}

func (m *model) buildOrderStatusForm() *huh.Form {
	return huh.NewForm(
		huh.NewGroup(
			huh.NewNote().Title(fmt.Sprintf("Commande #%d", m.targetOrderID)),
			huh.NewSelect[string]().
				Title("Nouveau statut").
				Options(
					huh.NewOption("pending", "pending"),
					huh.NewOption("paid", "paid"),
					huh.NewOption("shipped", "shipped"),
					huh.NewOption("delivered", "delivered"),
					huh.NewOption("cancelled", "cancelled"),
				).
				Value(&m.orderStatus),
		),
	).WithWidth(50)
}

func (m *model) updateOrderStatus(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok && key.String() == "esc" {
		m.screen = screenOrders
		return m, nil
	}
	form, cmd := m.orderStatusForm.Update(msg)
	if f, ok := form.(*huh.Form); ok {
		m.orderStatusForm = f
	}
	if m.orderStatusForm.State == huh.StateCompleted {
		orderID, status := m.targetOrderID, m.orderStatus
		m.screen = screenOrders
		m.loading = true
		return m, tea.Batch(cmd, m.doUpdateOrderStatus(orderID, status))
	}
	return m, cmd
}

func (m *model) doUpdateOrderStatus(orderID int64, status string) tea.Cmd {
	return func() tea.Msg {
		if err := m.api.UpdateOrderStatus(orderID, status); err != nil {
			return errMsg{err}
		}
		orders, err := m.api.ListAllOrders()
		if err != nil {
			return errMsg{err}
		}
		return ordersLoadedMsg{orders}
	}
}

func (m *model) buildCreateOrderForm() *huh.Form {
	m.orderEmail, m.orderProductID, m.orderQuantity = "", "", "1"
	return huh.NewForm(
		huh.NewGroup(
			huh.NewNote().Title("Nouvelle commande pour un client").
				Description("Un seul produit à la fois pour l'instant"),
			huh.NewInput().Title("Email du client").Value(&m.orderEmail).
				Validate(nonEmpty("email")),
			huh.NewInput().Title("ID du produit").Value(&m.orderProductID).
				Validate(validatePositiveInt),
			huh.NewInput().Title("Quantité").Value(&m.orderQuantity).
				Validate(validatePositiveInt),
		),
	).WithWidth(60)
}

func (m *model) updateCreateOrder(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok && key.String() == "esc" {
		m.screen = screenOrders
		return m, nil
	}
	form, cmd := m.createOrderForm.Update(msg)
	if f, ok := form.(*huh.Form); ok {
		m.createOrderForm = f
	}
	if m.createOrderForm.State == huh.StateCompleted {
		productID, _ := strconv.ParseInt(m.orderProductID, 10, 64)
		quantity, _ := strconv.Atoi(m.orderQuantity)
		req := api.CreateOrderRequest{
			Email: m.orderEmail,
			Items: []api.CreateOrderItemRequest{{ProductID: productID, Quantity: quantity}},
		}
		m.screen = screenOrders
		m.loading = true
		return m, tea.Batch(cmd, m.doCreateOrder(req))
	}
	return m, cmd
}

func (m *model) doCreateOrder(req api.CreateOrderRequest) tea.Cmd {
	return func() tea.Msg {
		if _, err := m.api.CreateOrderForUser(req); err != nil {
			return errMsg{err}
		}
		orders, err := m.api.ListAllOrders()
		if err != nil {
			return errMsg{err}
		}
		return ordersLoadedMsg{orders}
	}
}

func (m *model) updateUsers(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "esc":
			m.screen = screenMenu
			return m, nil
		case "a":
			m.addUserForm = m.buildAddUserForm()
			m.screen = screenAddUser
			return m, m.addUserForm.Init()
		case "e":
			if it, ok := m.users.SelectedItem().(userItem); ok {
				m.targetUserID = it.u.ID
				m.editEmail = it.u.Email
				m.editPassword = ""
				m.editOriginalAdmin = it.u.IsAdmin
				if it.u.IsAdmin {
					m.editAdminChoice = "oui"
				} else {
					m.editAdminChoice = "non"
				}
				m.editUserForm = m.buildEditUserForm()
				m.screen = screenEditUser
				return m, m.editUserForm.Init()
			}
		case "c":
			if it, ok := m.users.SelectedItem().(userItem); ok {
				m.loading = true
				return m, m.doConfirmUser(it.u.ID)
			}
		case "d":
			if it, ok := m.users.SelectedItem().(userItem); ok {
				m.loading = true
				return m, m.doDeleteUser(it.u.ID)
			}
		}
	}
	var cmd tea.Cmd
	m.users, cmd = m.users.Update(msg)
	return m, cmd
}

func (m *model) buildAddUserForm() *huh.Form {
	m.newUserEmail, m.newUserPass, m.newUserAdmin = "", "", false
	return huh.NewForm(
		huh.NewGroup(
			huh.NewNote().Title("Nouvel utilisateur"),
			huh.NewInput().Title("Email").Value(&m.newUserEmail).
				Validate(nonEmpty("email")),
			huh.NewInput().Title("Mot de passe").EchoMode(huh.EchoModePassword).
				Value(&m.newUserPass).
				Validate(func(s string) error {
					if len(s) < 4 {
						return fmt.Errorf("mot de passe trop court")
					}
					return nil
				}),
			huh.NewConfirm().Title("Administrateur ?").Value(&m.newUserAdmin),
		),
	).WithWidth(60)
}

func (m *model) updateAddUser(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok && key.String() == "esc" {
		m.screen = screenUsers
		return m, nil
	}
	form, cmd := m.addUserForm.Update(msg)
	if f, ok := form.(*huh.Form); ok {
		m.addUserForm = f
	}
	if m.addUserForm.State == huh.StateCompleted {
		req := api.CreateUserRequest{Email: m.newUserEmail, Password: m.newUserPass, IsAdmin: m.newUserAdmin}
		m.screen = screenUsers
		m.loading = true
		return m, tea.Batch(cmd, m.doCreateUser(req))
	}
	return m, cmd
}

func (m *model) doCreateUser(req api.CreateUserRequest) tea.Cmd {
	return func() tea.Msg {
		if _, err := m.api.CreateUser(req); err != nil {
			return errMsg{err}
		}
		users, err := m.api.ListUsers()
		if err != nil {
			return errMsg{err}
		}
		return usersLoadedMsg{users}
	}
}

func (m *model) buildEditUserForm() *huh.Form {
	return huh.NewForm(
		huh.NewGroup(
			huh.NewNote().Title(fmt.Sprintf("Modifier l'utilisateur #%d", m.targetUserID)),
			huh.NewInput().Title("Email").Value(&m.editEmail),
			huh.NewInput().Title("Nouveau mot de passe (vide pour ne pas changer)").
				EchoMode(huh.EchoModePassword).Value(&m.editPassword),
			huh.NewSelect[string]().
				Title("Administrateur ?").
				Options(
					huh.NewOption("oui", "oui"),
					huh.NewOption("non", "non"),
				).
				Value(&m.editAdminChoice),
		),
	).WithWidth(60)
}

func (m *model) updateEditUser(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok && key.String() == "esc" {
		m.screen = screenUsers
		return m, nil
	}
	form, cmd := m.editUserForm.Update(msg)
	if f, ok := form.(*huh.Form); ok {
		m.editUserForm = f
	}
	if m.editUserForm.State == huh.StateCompleted {
		var req api.UpdateUserRequest
		email := m.editEmail
		req.Email = &email
		if m.editPassword != "" {
			password := m.editPassword
			req.Password = &password
		}
		isAdmin := m.editAdminChoice == "oui"
		req.IsAdmin = &isAdmin
		userID := m.targetUserID
		m.screen = screenUsers
		m.loading = true
		return m, tea.Batch(cmd, m.doUpdateUser(userID, req))
	}
	return m, cmd
}

func (m *model) doUpdateUser(id int64, req api.UpdateUserRequest) tea.Cmd {
	return func() tea.Msg {
		if _, err := m.api.UpdateUser(id, req); err != nil {
			return errMsg{err}
		}
		users, err := m.api.ListUsers()
		if err != nil {
			return errMsg{err}
		}
		return usersLoadedMsg{users}
	}
}

func (m *model) doConfirmUser(id int64) tea.Cmd {
	return func() tea.Msg {
		if err := m.api.ConfirmUser(id); err != nil {
			return errMsg{err}
		}
		users, err := m.api.ListUsers()
		if err != nil {
			return errMsg{err}
		}
		return usersLoadedMsg{users}
	}
}

func (m *model) doDeleteUser(id int64) tea.Cmd {
	return func() tea.Msg {
		if err := m.api.DeleteUser(id); err != nil {
			return errMsg{err}
		}
		users, err := m.api.ListUsers()
		if err != nil {
			return errMsg{err}
		}
		return usersLoadedMsg{users}
	}
}

func (m *model) View() string {
	var body string
	help := "esc: retour · ctrl+c: quitter"

	switch m.screen {
	case screenAuth:
		body = m.authForm.View()
		help = "tab/flèches: naviguer · entrée: valider"
	case screenMenu:
		body = m.menuList.View()
		help = "↑/↓: naviguer · entrée: sélectionner · ctrl+c: quitter"
	case screenProducts:
		body = m.products.View()
		help = "a: ajouter · d: supprimer · esc: retour"
	case screenAddProduct:
		body = m.addProductForm.View()
	case screenOrders:
		body = m.orders.View()
		help = "s: changer le statut · n: nouvelle commande · esc: retour"
	case screenOrderStatus:
		body = m.orderStatusForm.View()
	case screenCreateOrder:
		body = m.createOrderForm.View()
	case screenUsers:
		body = m.users.View()
		help = "a: ajouter · e: modifier · c: confirmer · d: supprimer · esc: retour"
	case screenAddUser:
		body = m.addUserForm.View()
	case screenEditUser:
		body = m.editUserForm.View()
	}

	status := ""
	if m.loading {
		status = style.Subtitle.Render("Chargement…")
	} else if m.statusMsg != "" {
		if m.statusErr {
			status = style.StatusErr.Render("✗ " + m.statusMsg)
		} else {
			status = style.StatusOK.Render("✓ " + m.statusMsg)
		}
	}

	content := lipgloss.JoinVertical(lipgloss.Left, status, "", body)
	return style.Frame("Boutique - Espace admin", content, help)
}
