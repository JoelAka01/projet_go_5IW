# Exemple d'utilisateur

Ce document donne un exemple concret d'utilisateur et de parcours d'achat
(panier → paiement) pour tester l'application cliente (`cmd/client`).

## Compte de test

| Champ        | Valeur              |
| ------------ | ------------------- |
| Email        | `alice@example.com` |
| Mot de passe | `MotDePasse123!`    |
| Admin        | non                 |

Pour créer ce compte, lancer le client et choisir l'option de création de
compte :

```
1. Se connecter  2. Creer un compte
> 2
Email: alice@example.com
Mot de passe: MotDePasse123!
```

Il n'y a pas d'identifiant admin par défaut. Pour obtenir un compte
administrateur, créez d'abord un compte normal comme ci-dessus, puis
promouvez-le avec la commande de seed :

```bash
go run ./cmd/seed make-admin alice@example.com
```

## Exemple de panier

Un panier sauvegardé possède un identifiant métier au format `BSK-XXXXXX`
(ex: `BSK-1KH8E7`), affiché via l'option **2. Voir mon panier** :

```
Panier BSK-1KH8E7
#1 T-shirt x2 - 47.98 TTC (unite: 23.99 TTC)
#2 Mug x1 - 11.99 TTC (unite: 11.99 TTC)
Total TTC: 59.97 euros (livraison gratuite)
```

## Exemple de paiement (option 6 : Valider la commande)

Aucune connexion à Stripe n'est effectuée : les informations bancaires sont
uniquement validées localement (numéro via l'algorithme de Luhn, date
d'expiration future, CVC à 3 ou 4 chiffres).

| Champ                     | Valeur exemple        |
| ------------------------- | --------------------- |
| Numéro de carte           | `4111 1111 1111 1111` |
| Date d'expiration (MM/AA) | `12/28`               |
| CVC                       | `123`                 |

Déroulé dans le CLI :

```
Paiement du panier BSK-1KH8E7 - total: 59.97 euros
Numero de carte: 4111111111111111
Date d'expiration (MM/AA): 12/28
CVC: 123
Commande #4 creee (panier BSK-1KH8E7), total: 59.97 euros, carte terminant par 1111
```

> Remarque : le numéro `4111 1111 1111 1111` est un numéro de test classique
> (valide selon Luhn) ; aucune vraie carte n'est débitée, aucune donnée
> bancaire complète n'est enregistrée en base (seuls les 4 derniers chiffres
> le sont, via `internal/store/payments.go`).

## Exemples d'erreurs de validation

| Saisie                      | Erreur retournée           |
| --------------------------- | -------------------------- |
| Numéro `1234` (échoue Luhn) | `numero de carte invalide` |
| Date déjà passée (`01/20`)  | `carte expiree`            |
| CVC `12` (trop court)       | `CVC invalide`             |
