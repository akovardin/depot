# Depot

Менеджер репозитория андроидовских и iOS билиотек. Статья на [kovardin.ru](https://kovardin.ru/articles/microaps/depot/).

Мне понадобилось публиковать библиотеки для Android (и iOS но это потом). Очевидно для этого нужен maven репозиторий.

Maven репозиторий устроен довольно просто. Самый простой способ выложить библиотеку - опубликовать ее локально и скопировать в публичную папку на сервере. Тут нет никакой магии, все очевидно и в 90% случаев этого достаточно.

Но очень хочется иметь возможность запустить команду из грейдла и получить опубликованную библиотеку на сервере. Нужна возможность удалять артефакты, добавлять какое нибудь описание артефакта и версии. Желательно красиво выводить список артефактов на веб-страничке

Конечно, задача не новая. Сейчас есть много сервисов, которые можно установить на свой сервер и паблишить сколько угодно библиотек:

- [JFrog](https://jfrog.com/) - классика, которая используется в большинстве больших компаний. Но вам такой комбаин навряд ли понадобиться
- [nexus](https://help.sonatype.com/en/product-information.html) - это самая универсальная штука.Тут ксть вообще все и даже свой собственный docker hub
- [gitflic](https://docs.gitflic.space/registry/maven_registry) - да, можно не заморачиваться с селфхостед, а выкладывать все в реджистри на gitflic  
- [github](https://docs.github.com/en/actions/use-cases-and-examples/publishing-packages/publishing-java-packages-with-gradle) - тож самое как и c gitflic
- [reposilite](https://reposilite.com/) - вот это очень классный аналог. Простой и понятный сервис который позволяет опубликовать библиотеки одной gradle командой. Есть подготовленный docker контейнер. Но! У него один критический недостаток. Он Сделан не мной

В результате моего небольшого исследования я написал селфхотстед сервис - [depot](https://gitflic.ru/project/kovardin/depot). Построен сервис на базе [PocketBase](https://pocketbase.io/) с несколькими дополнительными ручками и главной страницей со списком репозиториев

## Структура данных

Для мой задачи нужно всего две коллекции - артефакты и версии. Артефакты описывают саму библиотеку, а версии и так понятно что описывают.

![](screens/artifacts.png)

Для артефакта нам нужно:
- name - название библиотеки
- description - описание библиотеки
- group - неймспес для библиотеки, например ru.kovardin
- type - можно выбрать ios или android
- last - дата последнего обновления
- enabled - доступность библиотеки

Поля будут заполняться автоматически при публикации библиотеки. По умолчанию библиотеки будут доступны и enabled будет выставлен в `true`. Поле description нужно заполнять руками для каждого артефакта

С версиями тоже никаких сложностей

![](screens/versions.png)

Для артефакта нам нужно:
- name - название библиотеки
- notes - описание что изменилось в этой версии
- version - версия
- enable - доступна ли версия для установки

Notes - нужно заполнять для каждой версии самостоятельно

Создав две описанные выше коллекции, мы сделали половину сервиса для публикации библиотек. Осталось добавить логику загрузки и красивый список репозиториев

## Загрузка артефактов

Перед тем как начать загружать артефакты, нужно разобраться как добавить репозиторий в настройки Android проекта

В файле build.gradle для модуля который хотим опубликовать(например, somelib) добавляем код

```gradle
repositories {
    // ...
    maven {
        Properties properties = new Properties()
        properties.load(project.rootProject.file('local.properties').newDataInputStream())

        def uploadToken = properties.getProperty('uploadToken')

        name = "depot"
        url = "https://depot.kovardin.ru/packages"
        credentials(HttpHeaderCredentials) {
            name = "Authorization"
            value =  uploadToken
        }
        authentication {
            header(HttpHeaderAuthentication)
        }
    }
}
```

Конечно, мы не хотим дать всем возможность опубликовать что угодно в нашем реджестри, поэтому мы будем указывать токен для авторизации. Сам токен указан в файле `local.properties`, а тут мы получаем его из файла и передаем в заголовок `Authorization`. Чуть позже укажу как получить эту самую авторизацию

- depot.kovardin.ru/packages - это адрес моего реджистри
- depot - название репозитория, может быть любое. От этого название будет зависеть название gradle команды для публикации 

Указав все необходимые настройки, в gradle появиться команда вида:

```
./gradlew somelib:publishAllPublicationsToDepotRepository
```

После запуска этой команды, gradle соберет необходимые артефакты и отправит их на сервер в PUT запросе

Пример такого запроса 
```http
PUT /packages/ru/kovardin/somelib/0.1.1/somelib-0.1.1.aar
PUT /packages/ru/kovardin/somelib/maven-metadata.xml
...
```
А содержимое каждого файла прилетит в теле запроса. Для работы с такими запросами, создаем новую ручку `artifacts.Publish` и добавляем ее для обработки PUT запросов

```go
e.Router.PUT("/packages/*", artifacts.Publish, apis.RequireAdminAuth())
```
`apis.RequireAdminAuth()` - это миделваря, которая делает ручку доступной только для администратора.

В методе `artifacts.Publish` нужно получить название файла, его содержимое и сохранить все на сервере.

```go
file := strings.Replace(c.Request().URL.Path, "/packages/", "", -1)

defer c.Request().Body.Close()

uploadFolder := path.Join(h.settings.UploadFolder(""), filepath.Dir(file))
uploadFile := path.Join(uploadFolder, filepath.Base(file))

if err := os.MkdirAll(uploadFolder, os.ModePerm); err != nil {
    return err
}

dst, err := os.Create(uploadFile)
if err != nil {
    return err
}

defer dst.Close()

if _, err := io.Copy(dst, c.Request().Body); err != nil {
    return err
}
```

Нужно создать новый файл на сервере в локации `ru/kovardin/somelib/` и записать в него все содержимое из `c.Request().Body`

Теперь нужно создать сущности в базе. Сначала создаем запись о артефакте. Чтобы узнать название и группу артефакта - распарсим файл `maven-metadata.xml`. Кроме артефакта, в этом файле указаны названия доступных версий:

```xml
<?xml version="1.0" encoding="UTF-8"?>
<metadata>
  <groupId>ru.kovardin</groupId>
  <artifactId>boosty</artifactId>
  <versioning>
    <latest>0.1.1</latest>
    <release>0.1.1</release>
    <versions>
      <version>0.1.1</version>
    </versions>
    <lastUpdated>20240818151425</lastUpdated>
  </versioning>
</metadata>
```

Парсим файл и сохраняем все в базу:

```go
if filepath.Base(uploadFile) == "maven-metadata.xml" {
    metaFile := uploadFile
    data, err := os.ReadFile(metaFile)
    if err != nil {
        return err
    }

    metadata := Metadata{}
    if err := xml.Unmarshal(data, &metadata); err != nil {
        return err
    }

    artifact, _ := h.app.Dao().FindFirstRecordByFilter(
        "artifacts",
        "group = {:group} && name = {:name}",
        dbx.Params{"group": metadata.GroupId, "name": metadata.ArtifactId},
    )
    if artifact == nil {
        collection, err := h.app.Dao().FindCollectionByNameOrId("artifacts")
        if err != nil {
            return err
        }

        artifact = models.NewRecord(collection)
    }

    artifact.Set("name", metadata.ArtifactId)
    artifact.Set("group", metadata.GroupId)
    artifact.Set("type", "android")
    artifact.Set("enabled", true)

    if err := h.app.Dao().SaveRecord(artifact); err != nil {
        return err
    }
// ...
}
```

Осталось добавить в базу версии, указанные в файле `maven-metadata.xml`

```go
for _, ver := range metadata.Versioning.Versions.Version {
    version, _ := h.app.Dao().FindFirstRecordByFilter("versions",
        "name = {:name} && version = {:version} && artifact = {:artifact}",
        dbx.Params{"name": metadata.ArtifactId, "version": ver, "artifact": artifact.Id},
    )
    if version == nil {
        collection, err := h.app.Dao().FindCollectionByNameOrId("versions")
        if err != nil {
            return err
        }

        version = models.NewRecord(collection)
    }

    version.Set("name", metadata.ArtifactId)
    version.Set("version", ver)
    version.Set("artifact", artifact.Id)
    version.Set("enabled", true)

    if err := h.app.Dao().SaveRecord(version); err != nil {
        return err
    }
}
```

Отлично, новая версия библиотеки загружена на сервер. Нужно реализовать возможность скачивать эти версии. Gradle для скачивания отправляет два запроса `GET` и `HEAD`. Реализуем очень простые ручки, которые позволяют отдать любой файл по запросу

```go
e.Router.GET("/packages/*", apis.StaticDirectoryHandler(os.DirFS(settings.UploadFolder("")), true))
e.Router.HEAD("/packages/*", apis.StaticDirectoryHandler(os.DirFS(settings.UploadFolder("")), true))
```

Скачивать и устанавливать библиотеки могут все пользователи, поэтому тут нет миделвари `apis.RequireAdminAuth()`

## Авторизация

Теперь о том, как получить токен администратора для загрузки библиотеки. Для этого нужно оправить POST запрос на ручку `/api/admins/auth-with-password`

```http
POST https://127.0.0.1:8080/api/admins/auth-with-password
Content-Type: application/json

{
    "identity": "example@yandex.ru",
    "password": "password"
}
```

В ответ вернется JSON с полем token

```json
{
  "admin": {
    "id": "zf8ehix5ljg2fu5",
    "created": "2024-08-04 23:10:15.851Z",
    "updated": "2024-08-04 23:10:15.851Z",
    "avatar": 0,
    "email": "example@yandex.ru"
  },
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE3MjUyMDU1NTEsImlkIjoiemY4ZWhpeDVsamcyZnU1IiwidHlwZSI6ImFkbWluIn0.j7utkMnSHvd9jNLKCZJIoZ9SKhPt2Tcj1DuuIZCeUOo"
}
```

Этот token нужно использовать для авторизации при загрузке библиотеки

## Список артефактов

Уже сейчас достаточно функциональности, чтобы использовать сервис для публикации и использования библиотек в ваших Android проектах. Но добавим чуть-чуть красоты и выведем список библиотек на главной странице сервиса.

Для этого получаем список артефактов и последнюю версию:

```go
records, err := h.app.Dao().FindRecordsByFilter(
    "artifacts",
    "enabled = true",
    "-created",
    100,
    0,
)

if err != nil {
    h.app.Logger().Error("failed fetch artifacts", "err", err)
}

aa := []Artifact{}
for _, record := range records {
    versions, err := h.app.Dao().FindRecordsByFilter(
        "versions",
        "enabled = true && artifact = {:artifact}",
        "-created",
        1,
        0,
        dbx.Params{"artifact": record.Id},
    )

    if err != nil {
        h.app.Logger().Error("failed fetch versions", "err", err)

        continue
    }

    v := ""
    if len(versions) > 0 {
        v = versions[0].GetString("version")
    }

    aa = append(aa, Artifact{
        Name:        record.GetString("name"),
        Group:       record.GetString("group"),
        Description: record.GetString("description"),
        LastVersion: v,
    })
}

html, err := h.registry.LoadFS(views.FS,
    "layout.html",
    "home/home.html",
).Render(map[string]any{
    "artifacts": aa,
})
```

В результате получаем красивую страничку со списком всех доступных библиотек

![](screens/home.png)

Конечно, еще много всего нужно улучшить и добавить:

- Сделать докер образ для быстрой установки
- Добавить список всех доступных версий
- Доработать удаление артефактов
- Добавить возможность закрывать доступ к артефактам по тоглу enabled
- Переключать репозиторий в приватный публичный режим
- Добавить регистрацию пользователей
- Написать подробную документацию

Надеюсь, у меня хватит времени добраться до всех этих задач. А сейчас проект доступен по на gitflic - [depot](https://gitflic.ru/project/kovardin/depot)

## Ссылки

- [Исходники проекта depot](https://gitflic.ru/project/kovardin/depot)
- [Классный аналог для своего реджестри - reposilite](https://reposilite.com/)
- [PocketBase - фреймоворк для быстрой разработки сервисов](https://pocketbase.io/)