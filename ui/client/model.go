package client

import (
	"fmt"
	"strconv"

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
	screenCatalog
	screenCart
	screenAddToCart
	screenUpdateCartItem
	screenCheckout
	screenOrders
	screenSearchInput
	screenSearchResults
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

type cartItem struct{ it api.CartItemDTO }

func (i cartItem) Title() string {
	return fmt.Sprintf("#%d %s x%d", i.it.ProductID, i.it.ProductName, i.it.Quantity)
}
func (i cartItem) Description() string {
	return fmt.Sprintf("%.2f€ TTC (unité: %.2f€)", float64(i.it.SubtotalCentsTTC)/100, float64(i.it.UnitPriceCentsTTC)/100)
}
func (i cartItem) FilterValue() string { return i.it.ProductName }

type orderItem struct{ o api.OrderDTO }

func (i orderItem) Title() string {
	return fmt.Sprintf("Commande #%d - %s", i.o.ID, i.o.Status)
}
func (i orderItem) Description() string {
	return fmt.Sprintf("%.2f€ · %d article(s)", float64(i.o.TotalCents)/100, len(i.o.Items))
}
func (i orderItem) FilterValue() string { return fmt.Sprintf("%d", i.o.ID) }

type errMsg struct{ err error }
type authDoneMsg struct{ resp *api.AuthResponse }
type productsLoadedMsg struct{ products []api.ProductDTO }
type cartLoadedMsg struct{ cart *api.CartDTO }
type ordersLoadedMsg struct{ orders []api.OrderDTO }
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

	catalog    list.Model
	cart       list.Model
	orders     list.Model
	searchList list.Model
	cartRef    string
	cartTotal  int64

	addToCartForm *huh.Form
	addProductID  int64
	addQuantity   string

	updateQtyForm   *huh.Form
	updateProductID int64
	updateQuantity  string

	checkoutForm *huh.Form
	cardNumber   string
	cardExpiry   string
	cardCVC      string

	searchForm  *huh.Form
	searchQuery string

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
	m.catalog = newList("Catalogue")
	m.cart = newList("Panier")
	m.orders = newList("Mes commandes")
	m.searchList = newList("Résultats de recherche")
	return m
}

func newMenuList() list.Model {
	items := []list.Item{
		simpleItem{"Voir le catalogue", "Parcourir les produits disponibles"},
		simpleItem{"Voir mon panier", "Articles ajoutés, quantités, total"},
		simpleItem{"Rechercher un produit", "Nom, catégorie, description ou prix"},
		simpleItem{"Voir mes commandes", "Historique de vos commandes"},
		simpleItem{"Quitter", "Fermer l'application"},
	}
	l := list.New(items, list.NewDefaultDelegate(), 0, 0)
	l.Title = "Menu client"
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
			huh.NewSelect[string]().
				Title("Que souhaitez-vous faire ?").
				Options(
					huh.NewOption("Se connecter", "login"),
					huh.NewOption("Créer un compte", "register"),
				).
				Value(&m.authMode),
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
		m.catalog.SetSize(msg.Width-4, listHeight)
		m.cart.SetSize(msg.Width-4, listHeight)
		m.orders.SetSize(msg.Width-4, listHeight)
		m.searchList.SetSize(msg.Width-4, listHeight)
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
		m.user = &msg.resp.User
		m.api.SetToken(msg.resp.Token)
		m.statusErr = false
		m.statusMsg = fmt.Sprintf("Connecté en tant que %s", m.user.Email)
		m.screen = screenMenu
		return m, nil

	case productsLoadedMsg:
		m.loading = false
		items := make([]list.Item, 0, len(msg.products))
		for _, p := range msg.products {
			items = append(items, productItem{p})
		}
		if m.screen == screenSearchResults {
			m.searchList.SetItems(items)
		} else {
			m.catalog.SetItems(items)
		}
		return m, nil

	case cartLoadedMsg:
		m.loading = false
		m.cartRef = msg.cart.Reference
		m.cartTotal = msg.cart.TotalCentsTTC
		items := make([]list.Item, 0, len(msg.cart.Items))
		for _, it := range msg.cart.Items {
			items = append(items, cartItem{it})
		}
		m.cart.SetItems(items)
		return m, nil

	case ordersLoadedMsg:
		m.loading = false
		items := make([]list.Item, 0, len(msg.orders))
		for _, o := range msg.orders {
			items = append(items, orderItem{o})
		}
		m.orders.SetItems(items)
		return m, nil
	}

	switch m.screen {
	case screenAuth:
		return m.updateAuth(msg)
	case screenMenu:
		return m.updateMenu(msg)
	case screenCatalog:
		return m.updateCatalog(msg)
	case screenCart:
		return m.updateCart(msg)
	case screenAddToCart:
		return m.updateAddToCart(msg)
	case screenUpdateCartItem:
		return m.updateUpdateCartItem(msg)
	case screenCheckout:
		return m.updateCheckout(msg)
	case screenOrders:
		return m.updateOrders(msg)
	case screenSearchInput:
		return m.updateSearchInput(msg)
	case screenSearchResults:
		return m.updateSearchResults(msg)
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
		email, password, mode := m.email, m.password, m.authMode
		m.authForm = m.buildAuthForm()
		return m, tea.Batch(cmd, m.doAuth(mode, email, password))
	}
	return m, cmd
}

func (m *model) doAuth(mode, email, password string) tea.Cmd {
	return func() tea.Msg {
		var (
			resp *api.AuthResponse
			err  error
		)
		if mode == "register" {
			resp, err = m.api.Register(email, password)
		} else {
			resp, err = m.api.Login(email, password)
		}
		if err != nil {
			return errMsg{err}
		}
		return authDoneMsg{resp}
	}
}

func (m *model) updateMenu(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok && key.String() == "enter" {
		idx := m.menuList.Index()
		switch idx {
		case 0:
			m.screen = screenCatalog
			m.loading = true
			return m, m.loadProducts()
		case 1:
			m.screen = screenCart
			m.loading = true
			return m, m.loadCart()
		case 2:
			m.screen = screenSearchInput
			m.searchForm = m.buildSearchForm()
			return m, m.searchForm.Init()
		case 3:
			m.screen = screenOrders
			m.loading = true
			return m, m.loadOrders()
		case 4:
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

func (m *model) loadCart() tea.Cmd {
	return func() tea.Msg {
		cart, err := m.api.GetCart()
		if err != nil {
			return errMsg{err}
		}
		return cartLoadedMsg{cart}
	}
}

func (m *model) loadOrders() tea.Cmd {
	return func() tea.Msg {
		orders, err := m.api.ListMyOrders()
		if err != nil {
			return errMsg{err}
		}
		return ordersLoadedMsg{orders}
	}
}

func (m *model) updateCatalog(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "esc":
			m.screen = screenMenu
			return m, nil
		case "a":
			if it, ok := m.catalog.SelectedItem().(productItem); ok {
				m.addProductID = it.p.ID
				m.addQuantity = "1"
				m.addToCartForm = m.buildAddToCartForm(it.p.Name)
				m.screen = screenAddToCart
				return m, m.addToCartForm.Init()
			}
		}
	}
	var cmd tea.Cmd
	m.catalog, cmd = m.catalog.Update(msg)
	return m, cmd
}

func (m *model) buildAddToCartForm(productName string) *huh.Form {
	return huh.NewForm(
		huh.NewGroup(
			huh.NewNote().Title("Ajouter au panier").Description(productName),
			huh.NewInput().
				Title("Quantité").
				Value(&m.addQuantity).
				Validate(validatePositiveInt),
		),
	).WithWidth(50)
}

func validatePositiveInt(s string) error {
	n, err := strconv.Atoi(s)
	if err != nil || n <= 0 {
		return fmt.Errorf("quantité invalide")
	}
	return nil
}

func (m *model) updateAddToCart(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok && key.String() == "esc" {
		m.screen = screenCatalog
		return m, nil
	}
	form, cmd := m.addToCartForm.Update(msg)
	if f, ok := form.(*huh.Form); ok {
		m.addToCartForm = f
	}
	if m.addToCartForm.State == huh.StateCompleted {
		qty, _ := strconv.Atoi(m.addQuantity)
		productID := m.addProductID
		m.screen = screenCatalog
		m.loading = true
		return m, tea.Batch(cmd, m.doAddToCart(productID, qty))
	}
	return m, cmd
}

func (m *model) doAddToCart(productID int64, qty int) tea.Cmd {
	return func() tea.Msg {
		if err := m.api.AddCartItem(productID, qty); err != nil {
			return errMsg{err}
		}
		return actionOKMsg{"Produit ajouté au panier."}
	}
}

func (m *model) updateCart(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "esc":
			m.screen = screenMenu
			return m, nil
		case "u":
			if it, ok := m.cart.SelectedItem().(cartItem); ok {
				m.updateProductID = it.it.ProductID
				m.updateQuantity = strconv.Itoa(it.it.Quantity)
				m.updateQtyForm = m.buildUpdateQtyForm(it.it.ProductName)
				m.screen = screenUpdateCartItem
				return m, m.updateQtyForm.Init()
			}
		case "d":
			if it, ok := m.cart.SelectedItem().(cartItem); ok {
				m.loading = true
				return m, m.doRemoveCartItem(it.it.ProductID)
			}
		case "c":
			m.screen = screenCheckout
			m.checkoutForm = m.buildCheckoutForm()
			return m, m.checkoutForm.Init()
		}
	}
	var cmd tea.Cmd
	m.cart, cmd = m.cart.Update(msg)
	return m, cmd
}

func (m *model) buildUpdateQtyForm(productName string) *huh.Form {
	return huh.NewForm(
		huh.NewGroup(
			huh.NewNote().Title("Modifier la quantité").Description(productName),
			huh.NewInput().
				Title("Nouvelle quantité (0 pour retirer)").
				Value(&m.updateQuantity).
				Validate(func(s string) error {
					n, err := strconv.Atoi(s)
					if err != nil || n < 0 {
						return fmt.Errorf("quantité invalide")
					}
					return nil
				}),
		),
	).WithWidth(50)
}

func (m *model) updateUpdateCartItem(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok && key.String() == "esc" {
		m.screen = screenCart
		return m, nil
	}
	form, cmd := m.updateQtyForm.Update(msg)
	if f, ok := form.(*huh.Form); ok {
		m.updateQtyForm = f
	}
	if m.updateQtyForm.State == huh.StateCompleted {
		qty, _ := strconv.Atoi(m.updateQuantity)
		productID := m.updateProductID
		m.screen = screenCart
		m.loading = true
		return m, tea.Batch(cmd, m.doUpdateCartItem(productID, qty))
	}
	return m, cmd
}

func (m *model) doUpdateCartItem(productID int64, qty int) tea.Cmd {
	return func() tea.Msg {
		if err := m.api.SetCartItemQuantity(productID, qty); err != nil {
			return errMsg{err}
		}
		cart, err := m.api.GetCart()
		if err != nil {
			return errMsg{err}
		}
		return cartLoadedMsg{cart}
	}
}

func (m *model) doRemoveCartItem(productID int64) tea.Cmd {
	return func() tea.Msg {
		if err := m.api.RemoveCartItem(productID); err != nil {
			return errMsg{err}
		}
		cart, err := m.api.GetCart()
		if err != nil {
			return errMsg{err}
		}
		return cartLoadedMsg{cart}
	}
}

func (m *model) buildCheckoutForm() *huh.Form {
	m.cardNumber, m.cardExpiry, m.cardCVC = "", "", ""
	return huh.NewForm(
		huh.NewGroup(
			huh.NewNote().Title("Paiement").Description(fmt.Sprintf("Panier %s - total: %.2f€", m.cartRef, float64(m.cartTotal)/100)),
			huh.NewInput().Title("Numéro de carte").Value(&m.cardNumber),
			huh.NewInput().Title("Expiration (MM/AA)").Value(&m.cardExpiry),
			huh.NewInput().Title("CVC").EchoMode(huh.EchoModePassword).Value(&m.cardCVC),
		),
	).WithWidth(50)
}

func (m *model) updateCheckout(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok && key.String() == "esc" {
		m.screen = screenCart
		return m, nil
	}
	form, cmd := m.checkoutForm.Update(msg)
	if f, ok := form.(*huh.Form); ok {
		m.checkoutForm = f
	}
	if m.checkoutForm.State == huh.StateCompleted {
		card := parseCard(m.cardNumber, m.cardExpiry, m.cardCVC)
		m.screen = screenMenu
		m.loading = true
		return m, tea.Batch(cmd, m.doCheckout(card))
	}
	return m, cmd
}

func parseCard(number, expiry, cvc string) api.CardDetailsDTO {
	month, year := 0, 0
	if _, err := fmt.Sscanf(expiry, "%d/%d", &month, &year); err == nil && year < 100 {
		year += 2000
	}
	return api.CardDetailsDTO{Number: number, ExpMonth: month, ExpYear: year, CVC: cvc}
}

func (m *model) doCheckout(card api.CardDetailsDTO) tea.Cmd {
	return func() tea.Msg {
		resp, err := m.api.Checkout(card)
		if err != nil {
			return errMsg{err}
		}
		return actionOKMsg{fmt.Sprintf(
			"Commande #%d créée (panier %s), total: %.2f€, carte terminant par %s",
			resp.Order.ID, resp.CartReference, float64(resp.Order.TotalCents)/100, resp.CardLast4,
		)}
	}
}

func (m *model) updateOrders(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok && key.String() == "esc" {
		m.screen = screenMenu
		return m, nil
	}
	var cmd tea.Cmd
	m.orders, cmd = m.orders.Update(msg)
	return m, cmd
}

func (m *model) buildSearchForm() *huh.Form {
	m.searchQuery = ""
	return huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Recherche (nom, description, catégorie ou prix)").
				Value(&m.searchQuery).
				Validate(func(s string) error {
					if s == "" {
						return fmt.Errorf("recherche vide")
					}
					return nil
				}),
		),
	).WithWidth(60)
}

func (m *model) updateSearchInput(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok && key.String() == "esc" {
		m.screen = screenMenu
		return m, nil
	}
	form, cmd := m.searchForm.Update(msg)
	if f, ok := form.(*huh.Form); ok {
		m.searchForm = f
	}
	if m.searchForm.State == huh.StateCompleted {
		query := m.searchQuery
		m.screen = screenSearchResults
		m.loading = true
		return m, tea.Batch(cmd, m.doSearch(query))
	}
	return m, cmd
}

func (m *model) doSearch(query string) tea.Cmd {
	return func() tea.Msg {
		products, err := m.api.SearchProducts(query)
		if err != nil {
			return errMsg{err}
		}
		return productsLoadedMsg{products}
	}
}

func (m *model) updateSearchResults(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok && key.String() == "esc" {
		m.screen = screenMenu
		return m, nil
	}
	var cmd tea.Cmd
	m.searchList, cmd = m.searchList.Update(msg)
	return m, cmd
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
	case screenCatalog:
		body = m.catalog.View()
		help = "a: ajouter au panier · esc: retour"
	case screenCart:
		body = lipgloss.JoinVertical(lipgloss.Left,
			m.cart.View(),
			fmt.Sprintf("Total TTC: %.2f€", float64(m.cartTotal)/100),
		)
		help = "u: modifier quantité · d: retirer · c: payer (checkout) · esc: retour"
	case screenAddToCart:
		body = m.addToCartForm.View()
	case screenUpdateCartItem:
		body = m.updateQtyForm.View()
	case screenCheckout:
		body = m.checkoutForm.View()
	case screenOrders:
		body = m.orders.View()
	case screenSearchInput:
		body = m.searchForm.View()
	case screenSearchResults:
		body = m.searchList.View()
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
	return style.Frame("Boutique - Espace client", content, help)
}
