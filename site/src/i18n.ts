// Every text of the page, in English and Brazilian Portuguese. Both objects
// have the same shape; code samples that are the same in both languages live
// in the components.

export type Lang = 'en' | 'pt';

const repo = 'https://github.com/lukascivil/trama';
const demo = 'https://github.com/lukascivil/trama-demo';

export const links = {
  repo,
  releases: `${repo}/releases`,
  demo,
  record: `${demo}/blob/main/changes/2026-10-31-rename-message-to-greeting.md`,
  metaharness: 'https://metaharness.tools/',
  mise: 'https://mise.jdx.dev',
  gh: 'https://cli.github.com',
  claude: 'https://code.claude.com',
};

const en = {
  meta: {
    title: 'trama: many repositories, one workspace your agent understands',
    description:
      'trama ties your repositories into one local workspace: it maps how they depend on each other with file:line evidence, runs them together and coordinates changes that span repositories.',
  },
  nav: {
    problem: 'Problem',
    how: 'How it works',
    strategies: 'Strategies',
    start: 'Get started',
    faq: 'FAQ',
    theme: 'Switch theme',
    language: 'Português',
    menu: 'Menu',
  },
  copy: 'Copy',
  copied: 'Copied',
  hero: {
    eyebrow: 'Local meta-harness for Claude Code',
    title: 'Many repositories.',
    titleAccent: 'One workspace your agent understands.',
    lead: 'trama ties your repositories into one workspace. It maps how they depend on each other, with file:line evidence for every link, runs them together and coordinates changes that cross repositories, in a safe order, with linked pull requests.',
    primary: 'Get started',
    secondary: 'View on GitHub',
    note: 'Open source · MIT · no server, no database, no SaaS',
    installTitle: 'In Claude Code',
  },
  problem: {
    kicker: 'The problem',
    title: 'Your agent sees one repository. Your system is many.',
    lead: 'A coding agent works inside one folder. A real feature crosses the frontend, the API, the gateway and a shared library. So you become the plumbing:',
    steps: [
      'Clone a dozen repositories and keep them up to date.',
      'Explain again, every session, which service calls which.',
      'Start each service by hand, with the right versions and ports.',
      'Find out who breaks when a field changes, with grep and memory.',
      'Open pull requests in the right order, and hope nobody merges the wrong one first.',
    ],
    outro:
      'A monorepo would solve part of this, but migrating a company to one is a project of its own. And sending your code to a hosted service to map it is often not an option.',
  },
  solution: {
    kicker: 'The answer',
    title: 'A workspace on top of the repositories you already have.',
    lead: 'A trama workspace is a plain git repository. It lists your repositories, clones them side by side and keeps a map of how they connect. The Claude Code plugin teaches the agent to read and use that map; a small Go CLI does the deterministic parts.',
    tree: [
      ['workspace.yaml', 'the repositories, their URLs and branches'],
      ['catalog/', 'one card per repository, every fact with file:line'],
      ['docs/graph.md', 'the dependency graph, generated'],
      ['AGENTS.md', 'how agents work here, with a repository table'],
      ['changes/', 'one record per change across repositories'],
      ['repos/', 'the clones, ignored by git'],
    ],
    points: [
      ['Nothing leaves your machine', 'No server, no database, no Docker. Files in git, reviewed like code.'],
      ['No migration', 'Each repository keeps its history, CI and permissions.'],
      ['Shared with the team', 'Push the workspace; a teammate clones it and runs trama sync.'],
    ],
  },
  how: {
    kicker: 'How it works',
    title: 'Map, run, measure the impact, change.',
    pillars: [
      {
        name: 'Map',
        command: '/trama:map',
        text: 'Reads each repository and writes its card: stack, how to run it, what it provides and what it consumes — down to the fields each endpoint accepts and returns, and the tables the repository owns, so you can ask who holds a piece of data. A consumer is linked to a provider only with proof. Without proof the link stays unknown, with its candidates, and trama validate checks every line a card cites.',
      },
      {
        name: 'Run',
        command: 'trama up',
        text: 'Starts every repository with the tool versions its card declares, through mise. Each line is prefixed with the repository name, and one Ctrl+C stops them all. setup, dev and test also work one repository at a time.',
      },
      {
        name: 'Impact',
        command: '/trama:impact',
        text: 'Tells what a change breaks before you touch anything: every affected repository with path:line, breaking or compatible, the tests that would not notice, and the safe order: expand, migrate, contract.',
      },
      {
        name: 'Change',
        command: '/trama:change',
        text: 'Writes a change record, waits for your approval, creates a branch in each repository, makes the change, runs each repository’s tests and opens linked pull requests in merge order. It never pushes to main and never merges.',
      },
    ],
  },
  relations: {
    kicker: 'Relations',
    title: 'Links with proof, not guesses.',
    lead: 'trama links two repositories only when it can show why. A wrong link misleads the agent more than a missing one.',
    kinds: [
      ['HTTP', 'Frontend to API, service to service, through a gateway: a config value that points to the provider’s host or port.'],
      ['Events', 'A topic the provider publishes and the consumer listens to.'],
      ['Packages', 'A shared library, by the package name the provider declares, private registries included.'],
      ['Declared', 'What the code cannot show, such as a route configured in a gateway, goes in workspace.yaml as a link.'],
    ],
    unknown: 'When there is no proof, the card says so: from: unknown, with the repositories that could be the provider.',
  },
  strategies: {
    kicker: 'Strategies',
    title: 'One workspace, or one per domain.',
    lead: 'A workspace is a folder and a git repository: cheap to create, easy to throw away. Pick the shape that fits how your company is organized.',
    options: [
      {
        name: 'Everything in one workspace',
        text: 'All repositories side by side, every relation in one graph. Good for a small or medium system, or to see the whole picture once.',
        best: 'Best for: one team, or a first map of everything.',
      },
      {
        name: 'One workspace per domain',
        text: 'A workspace per domain or team: payments, identity, catalog. Each stays small and focused. trama maps relations inside a workspace, so add the repositories a domain depends on, such as the gateway or the auth service, to each workspace that uses them.',
        best: 'Best for: many teams, or a system too big for one map.',
      },
    ],
    tip: 'Not sure? Start with one workspace and split it by domain when the map gets too big to be useful.',
  },
  start: {
    kicker: 'Get started',
    title: 'From an empty folder to a mapped system.',
    requirements: 'You need',
    steps: [
      { title: 'Install the plugin', where: 'In Claude Code' },
      { title: 'Create a workspace', where: 'In Claude Code, in an empty folder' },
      { title: 'Add your repositories', where: 'In Claude Code: clones and maps each one' },
      { title: 'Run everything', where: 'In the terminal, inside the workspace' },
      { title: 'Plan and make a change', where: 'In Claude Code' },
      { title: 'After the merges', where: 'In Claude Code or the terminal' },
    ],
    demoTitle: 'See it on a real example',
    demoText: 'trama-demo is a workspace with a Go API and a React app. Its change record shows a field renamed across both, with linked pull requests.',
    demoWorkspace: 'Example workspace',
    demoRecord: 'Example change record',
  },
  origin: {
    kicker: 'Origin',
    title: 'Built on the meta-harness idea.',
    text: 'A harness like Claude Code runs an agent inside one repository. A meta-harness is the layer above it: it knows the repositories, how they connect and how to change them together. trama is a local, open-source take on that idea, for teams that cannot send their code to a third party.',
    link: 'Read about meta-harnesses',
  },
  faq: {
    kicker: 'FAQ',
    title: 'Questions',
    items: [
      ['Does my code leave my machine?', 'No. trama has no server and no database. The workspace is a git repository you host wherever you want, and the clones stay in repos/ on your machine.'],
      ['Do I need to move to a monorepo?', 'No. Each repository keeps its own history, CI and permissions. The workspace sits beside them.'],
      ['Which agents does it work with?', 'The plugin is for Claude Code. The CLI is a standalone Go binary, and the workspace’s AGENTS.md describes the repositories to any agent that reads it.'],
      ['Does it work outside GitHub?', 'Mapping and running work with any git remote. Pull requests are opened with the GitHub CLI, so changes across repositories need GitHub today.'],
      ['Will it push or merge for me?', 'It never pushes to a repository’s main branch and never merges. The only commits it makes on its own are change records, on the workspace’s main branch.'],
      ['Is it ready?', 'It is a pre-release (v0.0.x). Mapping, running, impact and changes across repositories work today. A check at the start of each session and onboarding for new teammates come next.'],
    ],
  },
  footer: {
    tagline: 'Many repositories, one workspace.',
    license: 'MIT licensed',
    releases: 'Releases',
    demo: 'Example workspace',
  },
};

const pt: typeof en = {
  meta: {
    title: 'trama: muitos repositórios, um workspace que o seu agente entende',
    description:
      'A trama liga os seus repositórios num workspace local: mapeia como eles dependem uns dos outros com evidência arquivo:linha, roda todos juntos e coordena mudanças que atravessam repositórios.',
  },
  nav: {
    problem: 'Problema',
    how: 'Como funciona',
    strategies: 'Estratégias',
    start: 'Começar',
    faq: 'Perguntas',
    theme: 'Trocar tema',
    language: 'English',
    menu: 'Menu',
  },
  copy: 'Copiar',
  copied: 'Copiado',
  hero: {
    eyebrow: 'Meta-harness local para o Claude Code',
    title: 'Muitos repositórios.',
    titleAccent: 'Um workspace que o seu agente entende.',
    lead: 'A trama liga os seus repositórios num workspace só. Ela mapeia como eles dependem uns dos outros, com evidência arquivo:linha para cada ligação, roda todos juntos e coordena mudanças que atravessam repositórios, na ordem segura, com PRs ligados entre si.',
    primary: 'Começar',
    secondary: 'Ver no GitHub',
    note: 'Código aberto · MIT · sem servidor, sem banco, sem SaaS',
    installTitle: 'No Claude Code',
  },
  problem: {
    kicker: 'O problema',
    title: 'O seu agente enxerga um repositório. O seu sistema são vários.',
    lead: 'Um agente de código trabalha dentro de uma pasta. Uma funcionalidade de verdade passa pelo front-end, pela API, pelo gateway e por uma biblioteca compartilhada. Aí você vira o encanamento:',
    steps: [
      'Clona uma dúzia de repositórios e mantém todos atualizados.',
      'Explica de novo, a cada sessão, qual serviço chama qual.',
      'Sobe cada serviço na mão, com as versões e portas certas.',
      'Descobre quem quebra quando um campo muda, com grep e memória.',
      'Abre os PRs na ordem certa e torce para ninguém fazer merge do errado primeiro.',
    ],
    outro:
      'Um monorepo resolveria parte disso, mas migrar uma empresa para um monorepo é um projeto à parte. E mandar o seu código para um serviço externo mapear muitas vezes não é uma opção.',
  },
  solution: {
    kicker: 'A resposta',
    title: 'Um workspace por cima dos repositórios que você já tem.',
    lead: 'Um workspace da trama é um repositório git comum. Ele lista os seus repositórios, clona todos lado a lado e mantém um mapa de como eles se ligam. O plugin do Claude Code ensina o agente a ler e usar esse mapa; uma CLI pequena em Go faz a parte determinística.',
    tree: [
      ['workspace.yaml', 'os repositórios, suas URLs e branches'],
      ['catalog/', 'uma ficha por repositório, cada fato com arquivo:linha'],
      ['docs/graph.md', 'o grafo de dependências, gerado'],
      ['AGENTS.md', 'como os agentes trabalham aqui, com a tabela de repositórios'],
      ['changes/', 'um registro por mudança entre repositórios'],
      ['repos/', 'os clones, ignorados pelo git'],
    ],
    points: [
      ['Nada sai da sua máquina', 'Sem servidor, sem banco, sem Docker. Arquivos no git, revisados como código.'],
      ['Sem migração', 'Cada repositório mantém o seu histórico, CI e permissões.'],
      ['Compartilhado com o time', 'Envie o workspace; quem chega clona e roda trama sync.'],
    ],
  },
  how: {
    kicker: 'Como funciona',
    title: 'Mapear, rodar, medir o impacto, mudar.',
    pillars: [
      {
        name: 'Mapear',
        command: '/trama:map',
        text: 'Lê cada repositório e escreve a ficha dele: stack, como rodar, o que expõe e o que consome — até os campos que cada endpoint recebe e devolve, e as tabelas de que o repositório é dono, para você poder perguntar quem tem um dado. Um consumidor só é ligado a um fornecedor com prova. Sem prova, a ligação fica desconhecida, com os candidatos, e o trama validate confere cada linha que a ficha cita.',
      },
      {
        name: 'Rodar',
        command: 'trama up',
        text: 'Sobe todos os repositórios com as versões de ferramentas que cada ficha declara, via mise. Cada linha leva o nome do repositório, e um Ctrl+C para todos. setup, dev e test também funcionam um repositório por vez.',
      },
      {
        name: 'Impacto',
        command: '/trama:impact',
        text: 'Diz o que uma mudança quebra antes de você mexer em qualquer coisa: cada repositório afetado com caminho:linha, se quebra ou é compatível, os testes que não perceberiam e a ordem segura: expandir, migrar, contrair.',
      },
      {
        name: 'Mudar',
        command: '/trama:change',
        text: 'Escreve um registro da mudança, espera a sua aprovação, cria uma branch em cada repositório, faz a mudança, roda os testes de cada um e abre PRs ligados entre si, na ordem de merge. Nunca envia para a main e nunca faz merge.',
      },
    ],
  },
  relations: {
    kicker: 'Relações',
    title: 'Ligações com prova, não com palpite.',
    lead: 'A trama só liga dois repositórios quando consegue mostrar por quê. Uma ligação errada engana mais o agente do que uma que falta.',
    kinds: [
      ['HTTP', 'Front-end para API, serviço para serviço, passando por um gateway: um valor de configuração que aponta para o host ou a porta do fornecedor.'],
      ['Eventos', 'Um tópico que o fornecedor publica e o consumidor escuta.'],
      ['Pacotes', 'Uma biblioteca compartilhada, pelo nome de pacote que o fornecedor declara, inclusive em registros privados.'],
      ['Declaradas', 'O que o código não mostra, como uma rota configurada num gateway, vai no workspace.yaml como um vínculo.'],
    ],
    unknown: 'Quando não há prova, a ficha diz isso: from: unknown, com os repositórios que poderiam ser o fornecedor.',
  },
  strategies: {
    kicker: 'Estratégias',
    title: 'Um workspace só, ou um por domínio.',
    lead: 'Um workspace é uma pasta e um repositório git: barato de criar, fácil de jogar fora. Escolha o formato que combina com a organização da sua empresa.',
    options: [
      {
        name: 'Tudo num workspace só',
        text: 'Todos os repositórios lado a lado, todas as relações num grafo só. Bom para um sistema pequeno ou médio, ou para ver o quadro inteiro uma vez.',
        best: 'Melhor para: um time, ou um primeiro mapa de tudo.',
      },
      {
        name: 'Um workspace por domínio',
        text: 'Um workspace por domínio ou time: pagamentos, identidade, catálogo. Cada um fica pequeno e focado. A trama mapeia as relações dentro de um workspace, então inclua os repositórios de que um domínio depende, como o gateway ou o serviço de autenticação, em cada workspace que os usa.',
        best: 'Melhor para: vários times, ou um sistema grande demais para um mapa só.',
      },
    ],
    tip: 'Na dúvida, comece com um workspace só e divida por domínio quando o mapa ficar grande demais para ser útil.',
  },
  start: {
    kicker: 'Começar',
    title: 'De uma pasta vazia a um sistema mapeado.',
    requirements: 'Você precisa de',
    steps: [
      { title: 'Instale o plugin', where: 'No Claude Code' },
      { title: 'Crie um workspace', where: 'No Claude Code, numa pasta vazia' },
      { title: 'Adicione os seus repositórios', where: 'No Claude Code: clona e mapeia cada um' },
      { title: 'Rode tudo', where: 'No terminal, dentro do workspace' },
      { title: 'Planeje e faça uma mudança', where: 'No Claude Code' },
      { title: 'Depois dos merges', where: 'No Claude Code ou no terminal' },
    ],
    demoTitle: 'Veja num exemplo real',
    demoText: 'O trama-demo é um workspace com uma API em Go e um app React. O registro de mudança dele mostra um campo renomeado nos dois, com PRs ligados.',
    demoWorkspace: 'Workspace de exemplo',
    demoRecord: 'Registro de mudança de exemplo',
  },
  origin: {
    kicker: 'Origem',
    title: 'Construída sobre a ideia de meta-harness.',
    text: 'Um harness como o Claude Code roda um agente dentro de um repositório. Um meta-harness é a camada acima: conhece os repositórios, como eles se ligam e como mudá-los juntos. A trama é uma versão local e de código aberto dessa ideia, para times que não podem mandar o seu código para terceiros.',
    link: 'Leia sobre meta-harness',
  },
  faq: {
    kicker: 'Perguntas',
    title: 'Perguntas frequentes',
    items: [
      ['O meu código sai da minha máquina?', 'Não. A trama não tem servidor nem banco. O workspace é um repositório git hospedado onde você quiser, e os clones ficam em repos/ na sua máquina.'],
      ['Preciso migrar para um monorepo?', 'Não. Cada repositório mantém o seu histórico, CI e permissões. O workspace fica ao lado deles.'],
      ['Com quais agentes funciona?', 'O plugin é para o Claude Code. A CLI é um binário Go independente, e o AGENTS.md do workspace descreve os repositórios para qualquer agente que o leia.'],
      ['Funciona fora do GitHub?', 'Mapear e rodar funcionam com qualquer remoto git. Os PRs são abertos com o GitHub CLI, então mudanças entre repositórios precisam do GitHub por enquanto.'],
      ['Ela faz push ou merge por mim?', 'Nunca envia para a branch principal de um repositório e nunca faz merge. Os únicos commits que ela faz sozinha são os registros de mudança, na branch principal do workspace.'],
      ['Já está pronta?', 'É uma pré-versão (v0.0.x). Mapear, rodar, medir impacto e mudar vários repositórios já funcionam. Uma checagem no início de cada sessão e a entrada de quem chega no time vêm a seguir.'],
    ],
  },
  footer: {
    tagline: 'Muitos repositórios, um workspace.',
    license: 'Licença MIT',
    releases: 'Versões',
    demo: 'Workspace de exemplo',
  },
};

export const copy: Record<Lang, typeof en> = { en, pt };
