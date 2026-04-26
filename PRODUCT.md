# Product

## Register

product

## Users

Deux profils, parfois la même personne selon le projet :

- **Ingénieur IT interne / sysadmin** : valide une infra Proxmox (stockage, CPU) avant mise en production ou avant de la rendre disponible à un service métier. Contexte : journée de travail classique, double écran, pas de stress immédiat, exécute plusieurs jobs dans la semaine.
- **Consultant ou intégrateur externe** : déploie une infrastructure chez un client et doit fournir un rapport de validation à la livraison. Contexte : moment précis et important (réunion de recette, signature), souvent debout devant un client, parfois en visioconférence avec un écran partagé. Le rapport PDF n'est pas un sous-produit, c'est un livrable.

La densité d'usage va de quelques jobs par semaine à plusieurs par jour pendant une phase de delivery. Aucun utilisateur n'est nouveau au métier : tous comprennent IOPS, latence, débit, vCPU, storage pool, etc. Vocabulaire technique assumé.

## Product Purpose

Benchere mesure objectivement les performances d'une infrastructure Proxmox (stockage via elbencho, CPU via stress-ng) en provisionnant à la demande un cluster de workers, en lançant les benchmarks, puis en produisant un rapport.

L'outil existe parce que la mesure de performance d'infra est traditionnellement faite à la main, avec des scripts disparates, des résultats non comparables et aucun rapport présentable. Benchere standardise ce workflow : un job, des profils, un rapport, un verdict pass/fail face à des seuils.

Réussite = un consultant lance un job, attend, télécharge le PDF et le présente au client en réunion sans avoir à le retoucher.

## Brand Personality

**Confiant, opérationnel, sérieux.**

Voix : factuelle, concise, sans superlatifs. Ne survend pas ce que l'outil fait. Ne s'excuse pas non plus. Annonce un statut comme un système le ferait (`Provisionnement…`, `Tous les workers sont prêts.`, `Le job a échoué`).

Émotion visée : la même que celle d'ouvrir un terminal Linux familier, ou d'ouvrir Datadog avec un dashboard déjà configuré. Pas de surprise, pas de friction, pas de joie artificielle. La satisfaction vient de la lisibilité immédiate des chiffres et de la confiance qu'ils sont justes.

Le ton change légèrement dans le rapport PDF : reste factuel mais formellement plus soigné, parce qu'il sera lu par un client qui n'a pas le contexte interne.

## Anti-references

À éviter, par catégorie :

- **AI startup pastel** : gradients violet/rose, glassmorphism décoratif, hero cards centrées sur fond flou, "magic" copy. Vercel landing 2023 réplicated everywhere.
- **Cyber / crypto neon** : fond noir saturé + accents néon vert/bleu, gros chiffres animés, vibe "Bloomberg trader terminal" ou dashboard de mining.
- **Vieille entreprise / Bootstrap générique** : bleu corporate fade, tableaux denses sans hiérarchie, badges Bootstrap par défaut, look "intranet 2014".
- **Marketing SaaS productivity** : "Trusted by 10k teams", testimonial cards, big hero numbers décoratifs, three-column feature grids. Benchere n'est pas une landing page.

Aucune des références ci-dessus ne doit pouvoir être confondue avec Benchere au premier coup d'œil.

## Design Principles

1. **Le chiffre est le héros.** IOPS, latence, débit, statut. Tout le reste est chrome. Mono-tabulaire, alignement à droite, format compact (k, M). Aucun gradient ni shadow ne doit attirer l'œil avant la valeur.

2. **Densité Datadog, soin Linear.** Pas de compromis entre les deux. Le tableau d'historique est compact comme du Datadog ; les transitions, l'alignement et le typographique sont calibrés comme du Linear. Un sysadmin doit pouvoir scanner ; un consultant doit pouvoir présenter sans rougir.

3. **Le rapport PDF est l'interface la plus importante.** C'est l'artefact qui sort du système et qui survit au job. Il mérite autant d'attention que l'app vivante. Si le rapport ne supporte pas d'être imprimé et déposé sur la table d'un client, l'outil a manqué son but.

4. **L'orange est une signature, pas une décoration.** Couleur de marque réservée aux actions (CTA primaire), aux indicateurs actifs (route active, marqueur de progression) et aux ancres visuelles du rapport. Partout ailleurs : neutres légèrement tintés. Si tout est orange, plus rien ne l'est.

5. **Aucun théâtre.** Pas d'animation décorative, pas de "wow moment" gratuit, pas de skeuomorphisme. Une animation existe parce qu'elle informe (un dot qui pulse = job actif ; une barre qui se remplit = progression ; un fade entre routes = continuité). Sinon, immobile.

## Accessibility & Inclusion

- Cible WCAG AA : contraste 4.5:1 sur le texte courant, 3:1 sur les éléments structurels. Vérifié séparément en thème clair et sombre.
- Statut jamais véhiculé par la couleur seule : toujours couleur + texte + dot animé/forme.
- Focus ring visible (2px brand) sur tous les éléments interactifs ; jamais retiré globalement.
- Support `prefers-reduced-motion` : les pulses, fades de routes et transitions de progression doivent se neutraliser.
- Polices à charger via `font-display: swap` ; aucun texte invisible pendant le chargement.
- Le PDF doit rester lisible imprimé en noir et blanc (verdict pass/fail = pill avec mot, pas seulement la couleur).
- Pas d'authentification en V1 ; déploiement réseau interne uniquement.
