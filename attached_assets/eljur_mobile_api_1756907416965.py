#!/usr/bin/env python3
# -*- coding: utf-8 -*-

import requests
import json
import os
from typing import Dict, List, Any, Optional
from urllib.parse import quote, urlencode
from datetime import datetime

class EljurMobileAPI:
    """
    Полный API-клиент для мобильного приложения Эльжур.
    Реализует все запросы, обнаруженные в перехваченном трафике.
    """
    
    # Основной URL API
    BASE_URL = "https://eljur.gospmr.org/apiv3/"
    
    # Ключ разработчика
    DEV_KEY = "dd06cf484d85581e1976d93c639deee7"
    
    def __init__(self):
        """Инициализация клиента API."""
        self.auth_token = None
        self.student_id = None
        self.student_class = None
        self.domain = None
        self.cookies = {}
        self.session = requests.Session()
        self.results_dir = "api_results"
        
        # Создаем директорию для результатов, если она не существует
        if not os.path.exists(self.results_dir):
            os.makedirs(self.results_dir)
    
    def is_authenticated(self):
        """
        Проверяет, авторизован ли пользователь.
        
        Returns:
            bool: True, если пользователь авторизован, иначе False
        """
        return bool(self.auth_token and self.domain)
    
    def _get_headers(self):
        """Получение заголовков запроса"""
        headers = {}
        
        if self.cookies:
            cookies_str = "; ".join([f"{k}={v}" for k, v in self.cookies.items()])
            headers["cookie"] = cookies_str
        
        return headers
    
    def _get_common_params(self):
        """
        Формирование общих параметров для всех API-запросов.
        
        Returns:
            Dict с общими параметрами
        """
        params = {
            "devkey": self.DEV_KEY,
            "out_format": "json",
            "auth_token": self.auth_token or "",
            "vendor": "eljur"
        }
        
        return params
    
    def authenticate(self, login: str, password: str) -> Dict[str, Any]:
        """
        Аутентификация пользователя.
        
        Args:
            login: Логин пользователя (например, "Daniil_Melnik")
            password: Пароль пользователя
            
        Returns:
            Dict с информацией об аутентификации
        """
        print(f"Выполняется аутентификация пользователя: {login}...")
        
        # Параметры запроса для URL
        url_params = {
            "devkey": self.DEV_KEY,
            "out_format": "json",
            "auth_token": "",
            "vendor": "eljur"
        }
        
        # Данные формы для тела запроса
        form_data = {
            "login": login,
            "password": password
        }
        
        # Заголовки запроса
        headers = {
            "user-agent": "",
            "content-type": "application/x-www-form-urlencoded",
            "accept-encoding": "gzip"
        }
        
        # URL для аутентификации
        auth_url = f"{self.BASE_URL}auth"
        
        # Выполняем запрос
        response = self.session.post(
            auth_url, 
            params=url_params, 
            data=form_data, 
            headers=headers
        )
        
        print(f"URL запроса: {response.url}")
        print(f"Статус ответа: {response.status_code}")
        
        if response.status_code != 200:
            print(f"Ошибка аутентификации: {response.status_code}")
            return {"error": f"HTTP error {response.status_code}", "text": response.text}
        
        # Обрабатываем ответ
        try:
            data = response.json()
            
            # Проверяем статус ответа
            if data.get("response", {}).get("state") == 200:
                token = data["response"]["result"]["token"]
                school_domain = None
                for cookie in response.cookies:
                    if cookie.name == "school_domain":
                        school_domain = cookie.value
                        self.cookies[cookie.name] = cookie.value
                        print(f"Домен школы: {school_domain}")
                
                # Сохраняем token для использования в последующих запросах
                self.auth_token = token
                self.domain = school_domain
        
                print(f"Токен авторизации: {token[:10]}...")
                print(f"Домен школы: {school_domain}")
        
                # Сохраняем результат в файл
                self._save_result("auth", data)
                
                # Получаем информацию о пользователе (rules)
                self.get_rules()
                
                return {
                    "success": True,
                    "token": self.auth_token,
                    "domain": self.domain
                }
            else:
                error = data.get("response", {}).get("error", "Unknown error")
                print(f"Ошибка API при аутентификации: {error}")
                return {"error": error, "data": data}
                
        except ValueError as e:
            print(f"Ошибка декодирования JSON: {e}")
            return {"error": str(e), "text": response.text[:500]}
    
    def get_rules(self) -> Dict[str, Any]:
        """
        Получить правила и информацию о пользователе.
        
        Returns:
            Dict с информацией о пользователе и его правах
        """
        print("\nПолучение информации о пользователе (rules)...")
        
        # Проверяем, авторизован ли пользователь
        if not self.auth_token:
            print("Необходимо сначала авторизоваться.")
            return {"error": "Not authenticated"}
        
        # Параметры запроса
        params = {
            "devkey": self.DEV_KEY,
            "out_format": "json",
            "auth_token": self.auth_token,
            "vendor": "eljur"
        }
        
        # Заголовки запроса
        headers = {
            "user-agent": "",
            "accept-encoding": "gzip"
        }
        
        # URL для получения правил
        rules_url = f"{self.BASE_URL}getrules"
        
        # Выполняем запрос
        response = self.session.get(
            rules_url, 
            params=params, 
            headers=headers, 
            cookies=self.cookies
        )
        
        print(f"URL запроса: {response.url}")
        print(f"Статус ответа: {response.status_code}")
        
        if response.status_code != 200:
            print(f"Ошибка получения правил: {response.status_code}")
            return {"error": f"HTTP error {response.status_code}", "text": response.text}
        
        # Обрабатываем ответ
        try:
            data = response.json()
            
            # Проверяем статус ответа
            if data.get("response", {}).get("state") == 200:
                result = data["response"]["result"]
                
                # Извлекаем ID студента
                if result.get("id"):
                    self.student_id = result["id"]
                elif result.get("name"):
                    self.student_id = result["name"]
                
                # Извлекаем класс студента
                if "relations" in result:
                    if "students" in result["relations"]:
                        for student_key, student_info in result["relations"]["students"].items():
                            if student_info.get("class"):
                                self.student_class = student_info.get("class")
                                break
                    elif "groups" in result["relations"]:
                        for group_key, group_info in result["relations"]["groups"].items():
                            # Для класса ищем простой ключ без точек
                            if "." not in group_key:
                                self.student_class = group_key
                                break
                
                print(f"ID студента: {self.student_id}")
                print(f"Класс студента: {self.student_class}")
                
                # Сохраняем результат
                self._save_result("rules", data)
                
                return data
            else:
                error = data.get("response", {}).get("error", "Unknown error")
                print(f"Ошибка API при получении правил: {error}")
                return {"error": error, "data": data}
                
        except ValueError as e:
            print(f"Ошибка декодирования JSON: {e}")
            return {"error": str(e), "text": response.text[:500]}
    
    def get_extra_items_menu(self) -> Dict[str, Any]:
        """
        Получить дополнительные пункты меню.
        
        Returns:
            Dict с пунктами меню
        """
        print("\nПолучение дополнительных пунктов меню...")
        
        # Проверяем, авторизован ли пользователь
        if not self.auth_token:
            print("Необходимо сначала авторизоваться.")
            return {"error": "Not authenticated"}
        
        # Параметры запроса
        params = {
            "devkey": self.DEV_KEY,
            "out_format": "json",
            "auth_token": self.auth_token,
            "vendor": "eljur"
        }
        
        # Заголовки запроса
        headers = {
            "user-agent": "",
            "accept-encoding": "gzip"
        }
        
        # URL для получения дополнительных пунктов меню
        url = f"{self.BASE_URL}getextraitemsmenu"
        
        # Выполняем запрос
        response = self.session.get(
            url, 
            params=params, 
            headers=headers, 
            cookies=self.cookies
        )
        
        print(f"URL запроса: {response.url}")
        print(f"Статус ответа: {response.status_code}")
        
        # Обрабатываем ответ
        result = self._process_response(response, "extra_items_menu")
        return result
    
    def get_periods(self, weeks: bool = True, show_disabled: bool = True) -> Dict[str, Any]:
        """
        Получить периоды обучения.
        
        Args:
            weeks: Включать информацию о неделях
            show_disabled: Показывать отключенные периоды
            
        Returns:
            Dict с периодами обучения
        """
        print("\nПолучение периодов обучения...")
        
        # Проверяем, авторизован ли пользователь
        if not self.auth_token:
            print("Необходимо сначала авторизоваться.")
            return {"error": "Not authenticated"}
        
        # Параметры запроса
        params = {
            "weeks": "true" if weeks else "false",
            "show_disabled": "true" if show_disabled else "false",
            "devkey": self.DEV_KEY,
            "out_format": "json",
            "auth_token": self.auth_token,
            "vendor": "eljur"
        }
        
        # Заголовки запроса
        headers = {
            "user-agent": "",
            "accept-encoding": "gzip"
        }
        
        # URL для получения периодов обучения
        url = f"{self.BASE_URL}getperiods"
        
        # Выполняем запрос
        response = self.session.get(
            url, 
            params=params, 
            headers=headers, 
            cookies=self.cookies
        )
        
        print(f"URL запроса: {response.url}")
        print(f"Статус ответа: {response.status_code}")
        
        # Обрабатываем ответ
        result = self._process_response(response, "periods")
        return result
    
    def get_advertising_new(self) -> Dict[str, Any]:
        """
        Получить новую рекламу.
        
        Returns:
            Dict с рекламой
        """
        print("\nПолучение рекламы...")
        
        # Проверяем, авторизован ли пользователь
        if not self.auth_token:
            print("Необходимо сначала авторизоваться.")
            return {"error": "Not authenticated"}
        
        # Параметры запроса
        params = {
            "devkey": self.DEV_KEY,
            "out_format": "json",
            "auth_token": self.auth_token,
            "vendor": "eljur"
        }
        
        # Заголовки запроса
        headers = {
            "user-agent": "",
            "accept-encoding": "gzip"
        }
        
        # URL для получения рекламы
        url = f"{self.BASE_URL}getadvertisingnew"
        
        # Выполняем запрос
        response = self.session.get(
            url, 
            params=params, 
            headers=headers, 
            cookies=self.cookies
        )
        
        print(f"URL запроса: {response.url}")
        print(f"Статус ответа: {response.status_code}")
        
        # Обрабатываем ответ
        result = self._process_response(response, "advertising")
        return result
    
    def get_schedule(self, days: Optional[str] = None, class_id: Optional[str] = None) -> Dict[str, Any]:
        """
        Получить расписание занятий.
        
        Args:
            days: Период дней для расписания в формате 'YYYYMMDD-YYYYMMDD'
            class_id: ID класса, если отличается от класса пользователя
        
        Returns:
            Dict с расписанием
        """
        print("\nПолучение расписания...")
        
        # Проверяем, авторизован ли пользователь
        if not self.auth_token or not self.student_id or not self.student_class:
            print("Необходимо сначала авторизоваться и получить информацию о пользователе.")
            return {"error": "Not authenticated or incomplete user info"}
        
        # Если не указан период дней, используем текущую неделю
        if not days:
            # В реальном приложении тут должна быть логика вычисления текущей недели
            days = "20250512-20250518"  # Текущая неделя из перехваченного трафика
        
        # Если не указан класс, используем класс пользователя
        if not class_id:
            class_id = self.student_class
        
        # Параметры запроса
        params = {
            "student": self.student_id,
            "days": days,
            "class": class_id,
            "rings": "true",
            "devkey": self.DEV_KEY,
            "out_format": "json",
            "auth_token": self.auth_token,
            "vendor": "eljur"
        }
        
        # Заголовки запроса
        headers = {
            "user-agent": "",
            "accept-encoding": "gzip"
        }
        
        # URL для получения расписания
        url = f"{self.BASE_URL}getschedule"
        
        print(f"Запрос расписания с параметрами: {params}")
        
        # Выполняем запрос
        response = self.session.get(
            url, 
            params=params, 
            headers=headers, 
            cookies=self.cookies
        )
        
        print(f"URL запроса: {response.url}")
        print(f"Статус ответа: {response.status_code}")
        
        # Обрабатываем ответ
        result = self._process_response(response, f"schedule_{days}")
        return result
    
    def get_diary(self, days: Optional[str] = None) -> Dict[str, Any]:
        """
        Получить дневник.
        
        Args:
            days: Период дней для дневника в формате 'YYYYMMDD-YYYYMMDD'
        
        Returns:
            Dict с дневником
        """
        print("\nПолучение дневника...")
        
        # Проверяем, авторизован ли пользователь
        if not self.auth_token or not self.student_id:
            print("Необходимо сначала авторизоваться и получить информацию о пользователе.")
            return {"error": "Not authenticated or incomplete user info"}
        
        # Если не указан период дней, используем текущую неделю
        if not days:
            # В реальном приложении тут должна быть логика вычисления текущей недели
            days = "20250512-20250518"  # Текущая неделя из перехваченного трафика
        
        # Параметры запроса
        params = {
            "student": self.student_id,
            "days": days,
            "rings": "true",
            "devkey": self.DEV_KEY,
            "out_format": "json",
            "auth_token": self.auth_token,
            "vendor": "eljur"
        }
        
        # Заголовки запроса
        headers = {
            "user-agent": "",
            "accept-encoding": "gzip"
        }
        
        # URL для получения дневника
        url = f"{self.BASE_URL}getdiary"
        
        # Выполняем запрос
        response = self.session.get(
            url, 
            params=params, 
            headers=headers, 
            cookies=self.cookies
        )
        
        print(f"URL запроса: {response.url}")
        print(f"Статус ответа: {response.status_code}")
        
        # Обрабатываем ответ
        result = self._process_response(response, f"diary_{days}")
        return result
    
    def _process_response(self, response, file_prefix: str) -> Dict[str, Any]:
        """
        Обработка ответа API и сохранение результата в файл.
        
        Args:
            response: Ответ от requests
            file_prefix: Префикс для имени файла результата
        
        Returns:
            Dict с данными ответа или информацией об ошибке
        """
        if response.status_code != 200:
            error_data = {"error": f"HTTP error {response.status_code}", "text": response.text[:1000]}
            self._save_result(f"{file_prefix}_error", error_data)
            return error_data
        
        try:
            data = response.json()
            
            # Проверяем статус ответа
            if data.get("response", {}).get("state") == 200:
                print(f"Успешный ответ от API.")
                self._save_result(file_prefix, data)
                return data
            else:
                error = data.get("response", {}).get("error", "Unknown error")
                print(f"Ошибка API: {error}")
                self._save_result(f"{file_prefix}_api_error", data)
                return {"error": error, "data": data}
                
        except ValueError as e:
            error_data = {"error": str(e), "text": response.text[:1000]}
            print(f"Ошибка декодирования JSON: {e}")
            self._save_result(f"{file_prefix}_json_error", error_data)
            return error_data
    
    def _save_result(self, file_prefix: str, data: Dict[str, Any]) -> None:
        """
        Сохранить результат в JSON файл.
        
        Args:
            file_prefix: Префикс для имени файла
            data: Данные для сохранения
        """
        timestamp = datetime.now().strftime("%Y%m%d_%H%M%S")
        filename = f"{self.results_dir}/{file_prefix}_{timestamp}.json"
        
        with open(filename, 'w', encoding='utf-8') as f:
            json.dump(data, f, ensure_ascii=False, indent=2)
        
        print(f"Результат сохранен в файл: {filename}")

    def get_messages(self, folder="inbox", unread_only=False, limit=200, page=1, filter_str=""):
        """
        Получение списка сообщений из указанной папки.
        
        Args:
            folder: Папка с сообщениями ('inbox' - входящие, 'sent' - отправленные)
            unread_only: Показывать только непрочитанные сообщения
            limit: Ограничение количества сообщений (0 - без ограничения)
            page: Номер страницы
            filter_str: Строка фильтрации
            
        Returns:
            Dict с данными ответа или информацией об ошибке
        """
        if not self.is_authenticated():
            return {"error": "Пользователь не авторизован"}
        
        if not self.student_id:
            return {"error": "ID студента не найден"}
        
        # Параметры запроса
        unread_param = "yes" if unread_only else "no"
        params = {
            "folder": folder,
            "unreadonly": unread_param,
            "limit": limit,
            "page": page,
            "filter": filter_str
        }
        
        # Добавляем служебные параметры
        params.update(self._get_common_params())
        
        # Формируем URL запроса
        url = f"{self.BASE_URL}getmessages"
        
        print(f"Получение сообщений из папки {folder}...")
        print(f"URL запроса: {url}?{urlencode(params)}")
        
        # Выполняем запрос
        response = requests.get(
            url,
            params=params,
            headers=self._get_headers()
        )
        
        print(f"Статус ответа: {response.status_code}")
        
        # Обрабатываем ответ
        result = self._process_response(response, f"messages_{folder}_page{page}")
        
        # Возвращаем результат
        return result
    
    def get_message_details(self, message_id):
        """
        Получение деталей конкретного сообщения.
        
        Args:
            message_id: Идентификатор сообщения
            
        Returns:
            Dict с данными ответа или информацией об ошибке
        """
        if not self.is_authenticated():
            return {"error": "Пользователь не авторизован"}
        
        if not self.student_id:
            return {"error": "ID студента не найден"}
        
        # Параметры запроса
        params = {
            "id": message_id
        }
        
        # Добавляем служебные параметры
        params.update(self._get_common_params())
        
        # Формируем URL запроса - используем правильный эндпоинт
        url = f"{self.BASE_URL}getmessageinfo"
        
        print(f"Получение деталей сообщения с ID {message_id}...")
        print(f"URL запроса: {url}?{urlencode(params)}")
        
        # Выполняем запрос
        response = requests.get(
            url,
            params=params,
            headers=self._get_headers()
        )
        
        print(f"Статус ответа: {response.status_code}")
        
        # Обрабатываем ответ
        result = self._process_response(response, f"message_details_{message_id}")
        
        # Возвращаем результат
        return result
    
    def get_marks(self, period=None, start_date=None, end_date=None):
        """
        Получение оценок за выбранный период.
        
        Args:
            period: Номер четверти (1-4) или None для произвольного периода
            start_date: Начальная дата в формате "YYYYMMDD" для произвольного периода
            end_date: Конечная дата в формате "YYYYMMDD" для произвольного периода
            
        Returns:
            Dict с оценками за выбранный период
        """
        if not self.is_authenticated():
            return {"error": "Пользователь не авторизован"}
        
        if not self.student_id:
            return {"error": "ID студента не найден"}
            
        # Определяем даты периода на основе выбранной четверти
        if period is not None:
            # Даты четвертей из перехваченного трафика
            quarters = {
                1: ("20240903", "20241102"),  # Первая четверть
                2: ("20241111", "20241230"),  # Вторая четверть
                3: ("20250120", "20250322"),  # Третья четверть
                4: ("20250331", "20250524")   # Четвертая четверть
            }
            
            if period not in quarters:
                return {"error": f"Неверный номер четверти: {period}. Доступны номера 1-4."}
                
            start_date, end_date = quarters[period]
        elif start_date is None or end_date is None:
            # Если не указаны ни четверть, ни даты - используем текущую (четвертую) четверть
            start_date, end_date = "20250331", "20250524"
            
        # Параметры запроса
        params = {
            "student": self.student_id,
            "days": f"{start_date}-{end_date}"
        }
        
        # Добавляем служебные параметры
        params.update(self._get_common_params())
        
        # Формируем URL запроса
        url = f"{self.BASE_URL}getmarks"
        
        print(f"Получение оценок за период {start_date}-{end_date}...")
        print(f"URL запроса: {url}?{urlencode(params)}")
        
        # Выполняем запрос
        response = requests.get(
            url,
            params=params,
            headers=self._get_headers()
        )
        
        print(f"Статус ответа: {response.status_code}")
        
        # Обрабатываем ответ
        result = self._process_response(response, f"marks_{start_date}-{end_date}")
        
        # Возвращаем результат
        return result
        
    def get_message_receivers(self):
        """
        Получение списка доступных получателей сообщений.
        
        Returns:
            Dict с данными о доступных получателях сообщений
        """
        if not self.is_authenticated():
            return {"error": "Пользователь не авторизован"}
        
        if not self.student_id:
            return {"error": "ID студента не найден"}
        
        # Формируем URL запроса
        url = f"{self.BASE_URL}getmessagereceivers"
        
        # Добавляем служебные параметры
        params = self._get_common_params()
        
        print(f"Получение списка доступных получателей сообщений...")
        print(f"URL запроса: {url}?{urlencode(params)}")
        
        # Выполняем запрос
        response = requests.get(
            url,
            params=params,
            headers=self._get_headers()
        )
        
        print(f"Статус ответа: {response.status_code}")
        
        # Обрабатываем ответ
        result = self._process_response(response, "message_receivers")
        
        # Возвращаем результат
        return result
    
    def send_message(self, recipients, subject, text, attachments=None):
        """
        Отправка сообщения.
        
        Args:
            recipients: Список идентификаторов получателей
            subject: Тема сообщения
            text: Текст сообщения
            attachments: Список вложений (опционально)
            
        Returns:
            Dict с данными ответа или информацией об ошибке
        """
        if not self.is_authenticated():
            return {"error": "Пользователь не авторизован"}
        
        if not self.student_id:
            return {"error": "ID студента не найден"}
        
        # Формируем URL запроса
        url = f"{self.BASE_URL}sendmessage"
        
        # Служебные параметры в URL
        url_params = self._get_common_params()
        url_with_params = f"{url}?{urlencode(url_params)}"
        
        # Данные формы для отправки
        form_data = {
            "users_to": ",".join(recipients) if isinstance(recipients, list) else recipients,
            "subject": subject,
            "text": text
        }
        
        # Добавляем вложения, если они есть
        if attachments:
            for i, attachment in enumerate(attachments):
                form_data[f"attach{i+1}"] = attachment
        
        print(f"Отправка сообщения...")
        print(f"URL запроса: {url_with_params}")
        print(f"Получатели: {form_data['users_to']}")
        
        # Выполняем запрос
        response = requests.post(
            url_with_params,
            data=form_data,
            headers=self._get_headers()
        )
        
        print(f"Статус ответа: {response.status_code}")
        
        # Обрабатываем ответ
        result = self._process_response(response, "message_send")
        
        # Возвращаем результат
        return result

# Основная функция для запуска примера использования API
def main():
    print("=== Клиент API Эльжур (мобильное приложение) ===")
    
    # Создаем клиент API
    api = EljurMobileAPI()
    
    # Данные для авторизации
    login = input("Введите логин: ")
    password = input("Введите пароль: ")
    
    # Авторизация
    auth_result = api.authenticate(login, password)
    
    if auth_result.get("error"):
        print(f"Ошибка авторизации: {auth_result.get('error')}")
        return
    
    print("\nВыберите действие:")
    print("1. Получить расписание на текущую неделю")
    print("2. Получить дневник на текущую неделю")
    print("3. Получить периоды обучения")
    print("4. Получить дополнительные пункты меню")
    print("5. Просмотр входящих сообщений")
    print("6. Просмотр отправленных сообщений")
    print("7. Отправить сообщение")
    print("8. Просмотреть детали сообщения")
    print("9. Получить оценки по четвертям")
    print("10. Выполнить все запросы")
    print("0. Выход")
    
    choice = input("Введите номер действия: ")
    
    if choice == "1":
        api.get_schedule()
    elif choice == "2":
        api.get_diary()
    elif choice == "3":
        api.get_periods()
    elif choice == "4":
        api.get_extra_items_menu()
    elif choice == "5":
        api.get_messages(folder="inbox")
    elif choice == "6":
        api.get_messages(folder="sent")
    elif choice == "7":
        # Получаем список доступных получателей
        receivers_result = api.get_message_receivers()
        
        if receivers_result.get("error"):
            print(f"Ошибка получения списка получателей: {receivers_result.get('error')}")
            # Запрашиваем ID получателей вручную
            recipients = input("Введите ID получателей (через запятую): ")
        else:
            receivers = receivers_result.get("response", {}).get("result", {}).get("receivers", [])
            
            if not receivers:
                print("Список получателей пуст")
                recipients = input("Введите ID получателей (через запятую): ")
            else:
                print("\nДоступные получатели:")
                # Преобразуем список получателей в удобный формат
                for i, receiver in enumerate(receivers):
                    receiver_id = receiver.get("id", "")
                    name = receiver.get("name", "")
                    print(f"{i+1}. {name} (ID: {receiver_id})")
                
                # Запрашиваем выбор получателей
                choice_input = input("\nВыберите номера получателей (через запятую): ")
                
                try:
                    selected_indices = [int(idx.strip()) - 1 for idx in choice_input.split(',')]
                    selected_receivers = [receivers[idx].get("id", "") for idx in selected_indices if 0 <= idx < len(receivers)]
                    recipients = selected_receivers
                    print(f"Выбранные получатели: {', '.join([receivers[idx].get('name', '') for idx in selected_indices if 0 <= idx < len(receivers)])}")
                except (ValueError, IndexError):
                    print("Ошибка выбора получателей. Введите ID вручную.")
                    recipients = input("Введите ID получателей (через запятую): ")
                    recipients = recipients.split(',')
        
        # Запрашиваем тему и текст сообщения
        subject = input("Введите тему сообщения: ")
        text = input("Введите текст сообщения: ")
        
        # Отправляем сообщение
        if isinstance(recipients, str):
            recipients = recipients.split(',')
        api.send_message(recipients, subject, text)
    elif choice == "8":
        message_id = input("Введите ID сообщения: ")
        api.get_message_details(message_id)
    elif choice == "9":
        # Получение оценок по четвертям
        print("\nВыберите четверть:")
        print("1. Первая четверть (03.09.2024 - 02.11.2024)")
        print("2. Вторая четверть (11.11.2024 - 30.12.2024)")
        print("3. Третья четверть (20.01.2025 - 22.03.2025)")
        print("4. Четвертая четверть (31.03.2025 - 24.05.2025)")
        print("5. За весь учебный год")
        
        quarter_choice = input("\nВыберите номер четверти: ")
        
        if quarter_choice == "5":  # За весь учебный год
            api.get_marks(start_date="20240903", end_date="20250524")
        else:
            try:
                quarter = int(quarter_choice)
                if 1 <= quarter <= 4:
                    api.get_marks(period=quarter)
                else:
                    print(f"Неверный выбор: {quarter_choice}. Доступны номера 1-5.")
            except ValueError:
                print(f"Неверный выбор: {quarter_choice}. Должно быть число от 1 до 5.")
    elif choice == "10":
        # Выполняем все запросы последовательно
        api.get_periods()
        api.get_extra_items_menu()
        api.get_advertising_new()
        api.get_schedule()
        api.get_diary()
        api.get_messages(folder="inbox")
        api.get_messages(folder="sent")
        # Получаем оценки за текущую четверть
        api.get_marks(period=4)
    else:
        print("Выход из программы")

# Выполняем основную функцию, если файл запущен напрямую
if __name__ == "__main__":
    main()
